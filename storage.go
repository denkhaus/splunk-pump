package main

import (
	"github.com/djherbis/stow"

	"github.com/boltdb/bolt"
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

func (p *Storage) Put(msg Message) error {
	var key []byte
	Int64ToBytes(msg.Time.UnixNano(), key)
	p.store.Put(key, msg)
}

func (p *Storage) Open() error {
	db, err := bolt.Open(p.dbPath, 0600, nil)
	if err != nil {
		return err
	}
	p.db = db
	p.store = stow.NewJSONStore(db, []byte("messages"))
	return nil
}

func (p *Storage) Close() {
	if p.db != nil {
		return p.db.Close()
	}

	return nil
}
