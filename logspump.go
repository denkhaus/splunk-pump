package main

import (
	"sync"

	"github.com/fsouza/go-dockerclient"
	"github.com/juju/errors"
	"github.com/sirupsen/logrus"
)

type LogsPump struct {
	sync.Mutex
	pumps    map[string]*ContainerPump
	adapters map[string]AdapterCreateFn
	client   *docker.Client
	storage  *Storage
}

func (p *LogsPump) Run() error {
	client, err := docker.NewClient(getopt("DOCKER_HOST", "unix:///var/run/docker.sock"))
	if err != nil {
		return errors.Annotate(err, "new docker client")
	}
	p.client = client

	if err := p.storage.Open(); err != nil {
		return errors.Annotate(err, "open storage")
	}

	containers, err := p.client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return errors.Annotate(err, "list containers")
	}

	for _, cont := range containers {
		p.pumpLogs(&docker.APIEvents{
			ID:     cont.ID,
			Status: "start",
		}, "0")
	}

	events := make(chan *docker.APIEvents)
	err = p.client.AddEventListener(events)
	if err != nil {
		return errors.Annotate(err, "add event listener")
	}

	for event := range events {
		id := event.ID
		logger.Infof("received %s event for container %s", event.Status, normalID(id))

		switch event.Status {
		case "start", "restart":
			go p.pumpLogs(event, "all")
		}
	}

	return errors.New("docker event stream closed")
}

func (p *LogsPump) RegisterAdapter(createFn AdapterCreateFn, host string) {
	p.Lock()
	defer p.Unlock()
	p.adapters[host] = createFn
}

func (p *LogsPump) Shutdown() {
	logger.Infof("shutdown received")
	for id := range p.pumps {
		p.removeContainerPump(id)
	}
	p.storage.Close()
}

func (p *LogsPump) ensureContainerPump(container *Container) (*ContainerPump, error) {
	p.Lock()
	defer p.Unlock()

	id := container.ID
	pump, ok := p.pumps[id]
	if !ok {
		logger.Infof("create container pump for id %s", container.Id())
		pump = NewContainerPump(p.storage, container)

		adapters := []Adapter{}
		for host, fnc := range p.adapters {
			ad, err := fnc(host)
			if err != nil {
				return nil, errors.Annotate(err, "create new adapter")
			}

			logger.Infof("new adapter %s created", ad)
			adapters = append(adapters, ad)
		}

		pump.AddAdapters(adapters...)
		p.pumps[id] = pump
	}
	return pump, nil
}

func (p *LogsPump) removeContainerPump(id string) {
	p.Lock()
	defer p.Unlock()
	if pump, ok := p.pumps[id]; ok {
		pump.Close()
		delete(p.pumps, id)
		logger.Infof("removed container pump for id %s", normalID(id))
	}
}

func (p *LogsPump) pumpLogs(event *docker.APIEvents, tail string) {
	id := event.ID
	c, err := p.client.InspectContainer(id)
	if err != nil {
		logrus.Fatal(errors.Annotate(err, "inspect container"))
	}

	cont := (*Container)(c)
	if !cont.CanPump() {
		return
	}

	pump, err := p.ensureContainerPump(cont)
	if err != nil {
		logger.Errorf("ensure container pump for id %s: %s", cont.Id(), err)
		return
	}

	go func() {
		defer p.removeContainerPump(id)
		since, err := p.storage.GetLastLogTS(cont.Id())
		if err != nil {
			logger.Errorf("cant get last log ts for id %s: %s", cont.Id(), err)
			return
		}

		logger.Infof("started log feed for id %s", cont.Id())
		err = p.client.Logs(docker.LogsOptions{
			Container:    id,
			OutputStream: pump.outwr,
			ErrorStream:  pump.errwr,
			Stdout:       true,
			Stderr:       true,
			Follow:       true,
			Tail:         tail,
			Timestamps:   true,
			Since:        since,
		})
		if err != nil {
			logger.Errorf("terminated log feed for id %s with error %s", cont.Id(), err)
		} else {
			logger.Infof("stopped log feed for id %s", cont.Id())
		}
	}()
}

func NewLogsPump(storagePath string) *LogsPump {
	pump := &LogsPump{
		pumps:    make(map[string]*ContainerPump),
		adapters: make(map[string]AdapterCreateFn),
		storage:  NewStorage(storagePath),
	}
	return pump
}
