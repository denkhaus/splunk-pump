package main

import (
	"time"

	"github.com/boltdb/bolt"
	"github.com/djherbis/stow"
)

type Storage struct {
	dbPath string
	store  *stow.Store
	db     *bolt.DB
}

func NewStorage(dbPath string) *Storage {
	st := &Storage{dbPath: dbPath}
	return st
}

func (p *Storage) GetLastLogTS(containerId string) (int64, error) {
	var timeStamp int64
	err := p.store.Get(containerId, &timeStamp)
	if err == stow.ErrNotFound {
		logger.Warnf("last log ts for %s not found, use default", containerId)
		return time.Now().Add(-24 * time.Hour).Unix(), nil
	}

	return timeStamp, nil
}

func (p *Storage) PutLastLogTS(containerId string, timeStamp int64) error {
	err := p.store.Put(containerId, timeStamp)
	return err
}

func (p *Storage) Open() error {
	logger.Debug("open storage")
	db, err := bolt.Open(p.dbPath, 0600, nil)
	if err != nil {
		return err
	}
	p.db = db
	p.store = stow.NewJSONStore(db, []byte("container:logs:ts"))
	return nil
}

func (p *Storage) Close() error {
	if p.db != nil {
		logger.Debug("close storage")
		err := p.db.Close()
		p.db = nil
		return err
	}

	return nil
}
