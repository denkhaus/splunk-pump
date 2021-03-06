package main

import (
	"fmt"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

type Container docker.Container

func (p *Container) CanPump() bool {
	if p.Config.Tty {
		logger.Debug("container ", p, " ignored: tty enabled")
		return false
	}

	if p.IsIgnored() {
		logger.Debug("container ", p, " ignored: environ ignore")
		return false
	}

	return true
}

func (p *Container) Id() string {
	return normalID(p.ID)
}

func (p *Container) NormalName() string {
	return normalName(p.Name)
}

func (p *Container) String() string {
	return fmt.Sprintf("%s - %s", p.Id(), p.NormalName())
}

func (p *Container) IsIgnored() bool {
	for _, kv := range p.Config.Env {
		kvp := strings.SplitN(kv, "=", 2)
		if len(kvp) == 2 && kvp[0] == "SPLUNKPUMP" && strings.ToLower(kvp[1]) == "ignore" {
			return true
		}
	}
	return false
}
