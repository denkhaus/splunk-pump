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
	adapters map[string]Adapter
	client   *docker.Client
	storage  *Storage
}

func (p *LogsPump) Run() error {
	client, err := docker.NewClient(getopt("DOCKER_HOST", "unix:///var/run/docker.sock"))
	if err != nil {
		return errors.Annotate(err, "new docker client")
	}
	p.client = client

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
		case "die":

		}
	}

	return errors.New("docker event stream closed")
}

func (p *LogsPump) RegisterAdapter(adapter Adapter) {
	p.Lock()
	defer p.Unlock()
	logger.Infof("register adapter %s", adapter)
	p.adapters[adapter.String()] = adapter
}

func (p *LogsPump) ensureContainerPump(container *Container) *ContainerPump {
	p.Lock()
	defer p.Unlock()

	id := container.ID
	pump, ok := p.pumps[id]
	if !ok {
		logger.Infof("create container pump for id %s", normalID(id))
		pump = NewContainerPump(p.storage, container)

		adapters := []Adapter{}
		for _, ad := range p.adapters {
			adapters = append(adapters, ad)
		}

		pump.AddAdapters(adapters...)
		p.pumps[id] = pump
	}
	return pump
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

	container := (*Container)(c)
	if !container.CanPump() {
		return
	}

	pump := p.ensureContainerPump(container)

	go func() {
		defer p.removeContainerPump(id)
		logger.Infof("started log feed for id %s", container.Id())
		err := p.client.Logs(docker.LogsOptions{
			Container:    id,
			OutputStream: pump.outwr,
			ErrorStream:  pump.errwr,
			Stdout:       true,
			Stderr:       true,
			Follow:       true,
			Tail:         tail,
		})
		if err != nil {
			logger.Errorf("terminated log feed for id %s with error %s", id, err)
		}else{
			logger.Infof("stopped log feed for id %s", container.Id())
		}		
	}()
}

func NewLogsPump(storagePath string) *LogsPump {
	pump := &LogsPump{
		pumps:    make(map[string]*ContainerPump),
		adapters: make(map[string]Adapter),
	}
	return pump
}
