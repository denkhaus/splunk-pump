package main

import (
	"bufio"
	"io"
	"strings"
	"sync"
	"time"
)

// Messages are log messages
type Message struct {
	Container *Container
	Source    string
	Data      string
	Time      time.Time
}

type containerPump struct {
	sync.Mutex
	container *Container
}

func newContainerPump(container *Container, stdout, stderr io.Reader) *containerPump {
	cp := &containerPump{
		container: container,
	}
	pump := func(source string, input io.Reader) {
		buf := bufio.NewReader(input)
		for {
			line, err := buf.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					debug("pump:", normalID(container.ID), source+":", err)
				}
				return
			}
			cp.persist(&Message{
				Data:      strings.TrimSuffix(line, "\n"),
				Container: container,
				Time:      time.Now(),
				Source:    source,
			})
		}
	}

	go pump("stdout", stdout)
	go pump("stderr", stderr)
	return cp
}

func (cp *containerPump) persist(msg *Message) {
	cp.Lock()
	defer cp.Unlock()

}
