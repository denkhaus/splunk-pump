package main

import "time"

// Messages are log messages
type Message struct {
	Container   *Container
	ContainerID string
	Source      string
	Data        string
	Time        time.Time
}

type Adapter interface {
	Stream(stream chan *Message)
	String() string
}
