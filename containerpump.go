package main

import (
	"bufio"
	"io"
	"strings"
	"sync"
	"time"
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
	cp.removeAdapters()
	logger.Infof("closed container pump for container %s", cp.container.Id())
}

func (cp *ContainerPump) removeAdapters() {
	cp.Lock()
	defer cp.Unlock()
	for ad, ch := range cp.adapters {
		close(ch)
		delete(cp.adapters, ad)
	}
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

		ch := make(chan *Message)
		cp.adapters[adapter] = ch
		go adapter.Stream(ch)
	}
}

func (cp *ContainerPump) Send(msg *Message) {
	cp.Lock()
	defer cp.Unlock()

	for _, ch := range cp.adapters {
		ch <- msg
	}

	go func() {
		id := msg.Container.Id()
		if err := cp.storage.PutLastLogTS(id, msg.Time.Unix()); err != nil {
			logger.Errorf("unable to store last log ts for %s", id)
		}
	}()
}

func NewContainerPump(storage *Storage, cont *Container) *ContainerPump {
	cp := &ContainerPump{
		container: cont,
		storage:   storage,
		adapters:  make(map[Adapter]chan *Message),
		wg:        &sync.WaitGroup{},
	}

	cp.outrd, cp.outwr = io.Pipe()
	cp.errrd, cp.errwr = io.Pipe()

	pump := func(source string, input io.Reader) {
		logger.Debugf("start container pump for source %s[%s]", source, cont)

		cp.wg.Add(1)
		defer func() {
			logger.Debugf("stopped container pump for source %s[%s]", source, cont)
			cp.wg.Done()
		}()

		buf := bufio.NewReader(input)

		for {
			line, err := buf.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					logger.Error("readstring: ", cont.Id(), source+":", err)
				}
				return
			}

			tsString := line[:30]
			tm, err := time.Parse(time.RFC3339Nano, tsString)
			if err != nil {
				logger.Errorf("unable to parse timestamp %q for container %s: %s",
					tsString, cont, err)
				return
			}

			cp.Send(&Message{
				Data:      strings.TrimSuffix(line[30:], "\n"),
				Container: cont,
				Time:      tm,
				Source:    source,
			})
		}
	}

	go pump("stdout", cp.outrd)
	go pump("stderr", cp.errrd)
	return cp
}
