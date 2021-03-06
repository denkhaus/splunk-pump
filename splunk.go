package main

import (
	"bytes"
	"encoding/json"
	"net"
	"os"
	"time"

	"github.com/juju/errors"
)

type SplunkAdapter struct {
	address    *net.TCPAddr
	connection *net.TCPConn
	queue      chan *Message
	done       chan bool
	hostName   string
}

type SplunkMessage struct {
	Time            time.Time `json:"time"`
	ContainerHost   string    `json:"containerHost"`
	ContainerSource string    `json:"containerSource"`
	ContainerName   string    `json:"containerName"`
	ContainerID     string    `json:"containerID"`
	Line            string    `json:"line"`
}

func NewSplunkAdapter(addrStr string) (Adapter, error) {
	if len(addrStr) == 0 {
		return nil, errors.New("splunk: address missing")
	}

	address, err := net.ResolveTCPAddr("tcp", addrStr)
	if err != nil {
		return nil, errors.Annotate(err, "splunk resolve address")
	}

	hostName, err := os.Hostname()
	if err != nil {
		return nil, errors.Annotate(err, "get hostname")
	}

	queue := make(chan *Message, 1024)
	done := make(chan bool, 1)

	adapter := &SplunkAdapter{
		address:  address,
		queue:    queue,
		done:     done,
		hostName: hostName,
	}

	if err = adapter.connect(); err != nil {
		return nil, errors.Annotate(err, "splunk connect")
	}

	go adapter.writer()
	return adapter, nil
}

func (p *SplunkAdapter) connect() error {
	connection, err := net.DialTCP("tcp", nil, p.address)
	if err != nil {
		return errors.Annotate(err, "splunk dialtcp")
	}

	if err = connection.SetKeepAlive(true); err != nil {
		return errors.Annotate(err, "splunk set keep alive")
	}

	p.connection = connection
	return nil
}

func (p *SplunkAdapter) disconnect() error {
	if p.connection == nil {
		return nil
	}

	return p.connection.Close()
}

func (p *SplunkAdapter) reconnectLoop() {
	p.disconnect()

	var err error

	for {
		select {
		case <-p.done:
			break
		default:
		}

		err = p.connect()
		if err == nil {
			break
		}

		logger.Errorf("Splunk reconnect failed: %s\n", err)
		time.Sleep(1 * time.Second)
	}
}

func (p *SplunkAdapter) writeData(b []byte) {
	for {
		bytesWritten, err := p.connection.Write(b)
		if err != nil {
			logger.Errorf("Failed to write to TCP connection: %s\n", err)
			p.reconnectLoop()
			return
		}

		//logger.Infof("Wrote %v...", string(b))
		b = b[bytesWritten:]
		if len(b) == 0 {
			break
		}
	}
}

func (p *SplunkAdapter) writer() {
	for msg := range p.queue {
		buf, err := p.buildMessage(msg)
		if err != nil {
			logger.Error(errors.Annotate(err, "build message"))
			return
		}
		p.writeData(buf.Bytes())
	}
}

func (p *SplunkAdapter) String() string {
	return "splunk"
}

func (p *SplunkAdapter) buildMessage(m *Message) (*bytes.Buffer, error) {
	var msg = SplunkMessage{
		Time:            m.Time,
		ContainerHost:   p.hostName,
		ContainerSource: m.Source,
		ContainerName:   m.Container.NormalName(),
		ContainerID:     m.Container.Id(),
		Line:            m.Data,
	}

	data, err := json.Marshal(&msg)
	if err != nil {
		return nil, errors.Annotate(err, "marshal")
	}

	buf := bytes.NewBuffer(data)
	buf.WriteString("\n")
	return buf, nil

}

func (p *SplunkAdapter) Stream(stream chan *Message) {
	for message := range stream {
		select {
		case p.queue <- message:
		default:
			logger.Warning("Channel is full! Dropping events :-(")
			continue
		}
	}

	p.Close()
}

func (p *SplunkAdapter) Close() {
	close(p.queue)
	p.disconnect()
	p.done <- true
}
