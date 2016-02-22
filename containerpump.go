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
	wg        *sync.WaitGroup
	run       bool
	storage   *Storage
	container *Container
}

func newContainerPump(storage *Storage, container *Container, stdout, stderr io.Reader) *containerPump {
	cp := &containerPump{
		container: container,
		storage:   storage,
		wg:        &sync.WaitGroup{},
		run:       true,
	}

	pump := func(source string, input io.Reader) {
		cp.wg.Add(1)
		buf := bufio.NewReader(input)
		for cp.run {
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
		cp.wg.Done()
	}

	go pump("stdout", stdout)
	go pump("stderr", stderr)
	return cp
}

func (cp *containerPump) Stop() {
	cp.Lock()
	defer cp.Unlock()
	cp.run = false
	cp.wg.Wait()
}

func (cp *containerPump) Persist(msg *Message) {
	cp.Lock()
	defer cp.Unlock()

}
