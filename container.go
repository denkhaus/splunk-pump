package main

import (
	"strings"

	"github.com/fsouza/go-dockerclient"
)

type Container docker.Container

func (p *Container) CanPump() bool {
	if p.Config.Tty {
		debug("pump:", p.ID, "ignored: tty enabled")
		return false
	}

	if p.IsIgnored() {
		debug("pump:", p.ID, "ignored: environ ignore")
		return false
	}

	return true
}

func (p *Container) Id() string {
	return normalID(p.ID)
}

func (p *Container) IsIgnored() bool {
	for _, kv := range p.Config.Env {
		kvp := strings.SplitN(kv, "=", 2)
		if len(kvp) == 2 && kvp[0] == "LOGSPOUT" && strings.ToLower(kvp[1]) == "ignore" {
			return true
		}
	}
	return false
}
