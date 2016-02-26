package main

import (
	"os"
	"testing"

	"github.com/juju/errors"
)

func TestStorage(t *testing.T) {

	var testTS int64 = 1456495560

	os.Remove("/tmp/test.db")

	storage := NewStorage("/tmp/test.db")

	if err := storage.Open(); err != nil {
		t.Error(errors.Annotate(err, "open storage"))
	}

	if err := storage.PutLastLogTS("test", testTS); err != nil {
		t.Error(errors.Annotate(err, "put ts"))
	}

	ts, err := storage.GetLastLogTS("test")
	if err != nil {
		t.Error(errors.Annotate(err, "get ts"))
	}

	if ts != testTS {
		t.Error("Expected %d, got %d", testTS, ts)
	}

	if err := storage.Stats(); err != nil {
		t.Error(errors.Annotate(err, "print stats"))
	}

	if err := storage.Close(); err != nil {
		t.Error(errors.Annotate(err, "close storage"))
	}
}
