package main

import (
	"bufio"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
)

type containerPump struct {
	sync.Mutex
	wg        *sync.WaitGroup
	storage   *Storage
	container *Container
	done      chan bool
	adapters  map[Adapter]chan *Message
}

func (cp *containerPump) Stop() {
	cp.Lock()
	defer cp.Unlock()
	logger.Infof("stopping container pump for container %s", cp.container.Id())

	cp.done <- true
	cp.done <- true
	cp.wg.Wait()
	for ad, ch := range cp.adapters {
		close(ch)
		delete(cp.adapters, ad)
	}
	close(cp.done)
	logger.Infof("container pump for container %s stopped", cp.container.Id())
}

func (cp *containerPump) AddAdapters(adapters ...Adapter) {
	cp.Lock()
	defer cp.Unlock()

	for _, adapter := range adapters {
		for ad, _ := range cp.adapters {
			if ad.String() == adapter.String() {
				break
			}
		}

		cp.adapters[adapter] = make(chan *Message)
	}
}

func (cp *containerPump) Send(msg *Message) {
	cp.Lock()
	defer cp.Unlock()

	spew.Dump(msg.Data)
	for _, ch := range cp.adapters {
		ch <- msg
	}
}

func NewContainerPump(storage *Storage, container *Container, stdout, stderr io.Reader) *containerPump {
	cp := &containerPump{
		container: container,
		storage:   storage,
		adapters:  make(map[Adapter]chan *Message),
		done:      make(chan bool, 2),
		wg:        &sync.WaitGroup{},
	}

	pump := func(source string, input io.Reader) {
		logger.Infof("start container pump for source %s[%s]", source, container.Id())

		cp.wg.Add(1)
		buf := bufio.NewReader(input)
		for {
			select {
			case <-cp.done:
				break
			default:
			}

			line, err := buf.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					debug("pump:", container.Id(), source+":", err)
				}
				return
			}
			cp.Send(&Message{
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
