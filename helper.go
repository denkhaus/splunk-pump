package main

import (
	"os"
	"time"
)

func getopt(name, dfault string) string {
	value := os.Getenv(name)
	if value == "" {
		value = dfault
	}
	return value
}
func Int64ToBytes(value int64, buffer []byte) {
	mask := int64(0xff)

	var b byte
	v := value
	for i := 0; i < 8; i++ {
		b = byte(v & mask)
		buffer[i] = b
		v = v >> 8
	}
}

func retry(fn func() (interface{}, error), cnt int, msg string) (res interface{}, err error) {
	ret := 1
	for ret <= cnt {
		if res, err = fn(); err == nil {
			return
		}

		logger.Debugf("retry [%s] wait %d secs", msg, ret)
		time.Sleep(time.Duration(ret) * time.Second)
		ret++
	}

	return
}

func BytesToInt64(buffer []byte) int64 {
	var v int64

	v = int64(buffer[7])
	for i := 6; i >= 0; i-- {
		v = v<<8 + int64(buffer[i])
	}
	return v
}

func normalName(name string) string {
	return name[1:]
}

func normalID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
