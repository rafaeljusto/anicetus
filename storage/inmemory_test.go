package storage_test

import (
	"sync"
	"testing"
	"time"

	"github.com/rafaeljusto/anicetus/v2"
	"github.com/rafaeljusto/anicetus/v2/storage"
)

func TestInMemory_lifecycle(t *testing.T) {
	fingerprint := anicetus.Fingerprint("test")

	s := storage.NewInMemory()

	if elected, err := s.Add(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !elected {
		t.Error("first Add should elect the fingerprint")
	}

	if elected, err := s.Add(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if elected {
		t.Error("second Add should not elect the fingerprint")
	}

	if ok, err := s.Processed(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if ok {
		t.Error("fingerprint should not be processed yet")
	}

	if err := s.Store(t.Context(), fingerprint, true); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ok, err := s.Processed(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !ok {
		t.Error("fingerprint should be processed")
	}

	if err := s.Remove(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ok, err := s.Processed(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if ok {
		t.Error("fingerprint should not be processed after removal")
	}

	// After removal the fingerprint is available for a new election.
	if elected, err := s.Add(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !elected {
		t.Error("Add after Remove should elect the fingerprint again")
	}
}

// TestInMemory_Add_concurrent is the regression test for the election race: no
// matter how many goroutines call Add for the same fingerprint at once, exactly
// one must be elected.
func TestInMemory_Add_concurrent(t *testing.T) {
	const goroutines = 512

	s := storage.NewInMemory()
	fingerprint := anicetus.Fingerprint("test")

	var (
		wg      sync.WaitGroup
		start   = make(chan struct{})
		elected int64
		mu      sync.Mutex
	)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := s.Add(t.Context(), fingerprint)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if ok {
				mu.Lock()
				elected++
				mu.Unlock()
			}
		}()
	}

	close(start)
	wg.Wait()

	if elected != 1 {
		t.Errorf("expected exactly 1 elected request, got %d", elected)
	}
}

// TestInMemory_Add_leaseExpiry ensures a fingerprint whose lease expired (e.g.
// its elected request crashed without cleaning up) is released for a new
// election instead of blocking forever.
func TestInMemory_Add_leaseExpiry(t *testing.T) {
	lease := 50 * time.Millisecond

	s := storage.NewInMemory(storage.WithLeaseTTL(lease))
	fingerprint := anicetus.Fingerprint("test")

	if elected, err := s.Add(t.Context(), fingerprint); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !elected {
		t.Fatal("first Add should elect the fingerprint")
	}

	if elected, err := s.Add(t.Context(), fingerprint); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if elected {
		t.Fatal("Add within the lease should not elect the fingerprint")
	}

	time.Sleep(2 * lease)

	if elected, err := s.Add(t.Context(), fingerprint); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !elected {
		t.Error("Add after the lease expired should elect the fingerprint again")
	}
}
