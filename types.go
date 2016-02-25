package main

import "time"

// Messages are log messages
type Message struct {
	Container *Container
	Source    string
	Data      string
	Time      time.Time
}

type Adapter interface {
	Stream(stream chan *Message)
	String() string
}

type AdapterCreateFn func(addrStr string) (Adapter, error)
