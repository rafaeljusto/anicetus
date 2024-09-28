package storage_test

import (
	"testing"

	"github.com/rafaeljusto/anicetus"
	"github.com/rafaeljusto/anicetus/storage"
)

func TestInMemory_lifecycle(t *testing.T) {
	fingerprint := anicetus.Fingerprint("test")

	storage := storage.NewInMemory()
	if ok, err := storage.Exists(fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if ok {
		t.Error("unexpected fingerprint exists")
	}

	if ok, err := storage.Processed(fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if ok {
		t.Error("unexpected fingerprint processed")
	}

	if err := storage.Store(fingerprint, false); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ok, err := storage.Exists(fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !ok {
		t.Error("fingerprint should exists")
	}

	if ok, err := storage.Processed(fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if ok {
		t.Error("fingerprint should not be processed")
	}

	if err := storage.Store(fingerprint, true); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ok, err := storage.Processed(fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !ok {
		t.Error("fingerprint should be processed")
	}

	if err := storage.Remove(fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ok, err := storage.Exists(fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if ok {
		t.Error("fingerprint should not exists")
	}
}
