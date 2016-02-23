package main

import (
	"io"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/fsouza/go-dockerclient"
	"github.com/juju/errors"
	"github.com/sirupsen/logrus"
)

type LogsPump struct {
	sync.Mutex
	pumps    map[string]*containerPump
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
			ID:     normalID(cont.ID),
			Status: "start",
		}, "0")
	}

	events := make(chan *docker.APIEvents)
	err = p.client.AddEventListener(events)
	if err != nil {
		return errors.Annotate(err, "add event listener")
	}

	for event := range events {
		id := normalID(event.ID)
		logger.Infof("received event %s from container %s", event.Status, id)

		switch event.Status {
		case "start", "restart":
			go p.pumpLogs(event, "all")
		case "die":
			go p.removeContainerPump(id)
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

func (p *LogsPump) createContainerPump(container *Container, stdout, stderr io.Reader) bool {
	p.Lock()
	defer p.Unlock()

	id := container.Id()
	if _, ok := p.pumps[id]; !ok {
		logger.Infof("create container pump for id %s", id)
		pump := NewContainerPump(p.storage, container, stdout, stderr)

		adapters := []Adapter{}
		for _, ad := range p.adapters {
			adapters = append(adapters, ad)
		}

		pump.AddAdapters(adapters...)
		p.pumps[id] = pump
		return true
	}
	return false
}

func (p *LogsPump) removeContainerPump(id string) {
	p.Lock()
	defer p.Unlock()
	if pump, ok := p.pumps[id]; ok {
		pump.Stop()
		logger.Infof("remove container pump for id %s", id)
		delete(p.pumps, id)
	}
}

func (p *LogsPump) pumpLogs(event *docker.APIEvents, tail string) {

	spew.Dump(event.ID)

	id := normalID(event.ID)
	c, err := p.client.InspectContainer(id)
	if err != nil {
		logrus.Fatal(errors.Annotate(err, "inspect container"))
	}

	container := (*Container)(c)
	if !container.CanPump() {
		return
	}

	outrd, outwr := io.Pipe()
	errrd, errwr := io.Pipe()

	if p.createContainerPump(container, outrd, errrd) {
		logger.Infof("container pump for container %s started", id)

		go func() {
			err := p.client.Logs(docker.LogsOptions{
				Container:    id,
				OutputStream: outwr,
				ErrorStream:  errwr,
				Stdout:       true,
				Stderr:       true,
				Follow:       true,
				Tail:         tail,
			})
			if err != nil {
				logger.Errorf("container pump for container %s terminated with error %s", id, err)
			}

			outwr.Close()
			errwr.Close()
			p.removeContainerPump(id)
		}()
	}
}

func NewLogsPump(storagePath string) *LogsPump {
	pump := &LogsPump{
		pumps:    make(map[string]*containerPump),
		adapters: make(map[string]Adapter),
	}
	return pump
}
