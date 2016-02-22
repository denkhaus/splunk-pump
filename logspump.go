package main

import (
	"io"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/juju/errors"
)

type LogsPump struct {
	sync.Mutex
	pumps  map[string]*containerPump
	client *docker.Client
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
		}, false)
	}

	events := make(chan *docker.APIEvents)
	err = p.client.AddEventListener(events)
	if err != nil {
		return errors.Annotate(err, "add event listener")
	}

	for event := range events {
		id := normalID(event.ID)
		debug("pump: event:", id, event.Status)
		switch event.Status {
		case "start", "restart":
			go p.pumpLogs(event, true)
		case "die":
			go p.removeContainerPump(id)
		}
	}

	return errors.New("docker event stream closed")
}

func (p *LogsPump) createContainerPump(container *Container, stdout, stderr io.Reader) {
	p.Lock()
	defer p.Unlock()

	id := container.Id()
	if _, ok := p.pumps[id]; !ok {
		p.pumps[id] = newContainerPump(container, stdout, stderr)
	}
}

func (p *LogsPump) removeContainerPump(id string) {
	p.Lock()
	defer p.Unlock()
	if _, ok := p.pumps[id]; ok {
		delete(p.pumps, id)
	}
}

func (p *LogsPump) pumpLogs(event *docker.APIEvents, backlog bool) {
	id := normalID(event.ID)
	c, err := p.client.InspectContainer(id)
	if err != nil {
		logrus.Fatal(errors.Annotate(err, "inspect container"))
	}

	container := (*Container)(c)
	if !container.CanPump() {
		return
	}

	var tail string
	if backlog {
		tail = "all"
	} else {
		tail = "0"
	}

	outrd, outwr := io.Pipe()
	errrd, errwr := io.Pipe()

	p.createContainerPump(container, outrd, errrd)
	debug("pump:", id, "started")

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
			debug("pump:", id, "stopped:", err)
		}
		outwr.Close()
		errwr.Close()
		p.removeContainerPump(id)
	}()
}

func NewLogsPump() *LogsPump {
	pump := &LogsPump{
		pumps: make(map[string]*containerPump),
	}
	return pump
}
