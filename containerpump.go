package main

import (
	"bufio"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
)

type ContainerPump struct {
	sync.Mutex
	wg        *sync.WaitGroup
	storage   *Storage
	container *Container
	outrd     *io.PipeReader
	errrd     *io.PipeReader
	outwr     *io.PipeWriter
	errwr     *io.PipeWriter
	adapters  map[Adapter]chan *Message
}

func (cp *ContainerPump) Close() {
	cp.outwr.Close()
	cp.errwr.Close()
	cp.wg.Wait()

	cp.Lock()
	defer cp.Unlock()
	for ad, ch := range cp.adapters {
		close(ch)
		delete(cp.adapters, ad)
	}
	logger.Infof("closed container pump for container %s", cp.container.Id())
}

func (cp *ContainerPump) AddAdapters(adapters ...Adapter) {
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

func (cp *ContainerPump) Send(msg *Message) {
	cp.Lock()
	defer cp.Unlock()

	spew.Dump(msg.Data)
	//for _, ch := range cp.adapters {
	//	ch <- msg
	//}
}

func NewContainerPump(storage *Storage, container *Container) *ContainerPump {
	cp := &ContainerPump{
		container: container,
		storage:   storage,
		adapters:  make(map[Adapter]chan *Message),
		wg:        &sync.WaitGroup{},
	}

	cp.outrd, cp.outwr = io.Pipe()
	cp.errrd, cp.errwr = io.Pipe()

	pump := func(source string, input io.Reader) {
		logger.Infof("start container pump for source %s[%s]", source, container.Id())

		cp.wg.Add(1)
		defer func() {
			logger.Infof("stopped container pump for source %s[%s]", source, container.Id())
			cp.wg.Done()
		}()

		buf := bufio.NewReader(input)

		for {
			line, err := buf.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					logger.Errorf("readstring:", container.Id(), source+":", err)
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
	}

	go pump("stdout", cp.outrd)
	go pump("stderr", cp.errrd)
	return cp
}
