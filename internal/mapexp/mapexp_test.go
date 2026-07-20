package mapexp

import (
	"testing"
	"time"
)

// TestExpirationQueue_renewKeepsOrder is the regression test for renew breaking
// the ascending-expiration ordering that purge relies on. After a stale item is
// renewed it must move behind the still-expired items, otherwise purge stops
// scanning at the renewed item and never reclaims the expired ones.
func TestExpirationQueue_renewKeepsOrder(t *testing.T) {
	ttl := 200 * time.Millisecond
	q := newExpirationQueue[string](ttl)

	q.add("a")
	q.add("b")

	// Let both entries expire, then renew "a" so it is no longer expired while
	// "b" still is.
	time.Sleep(2 * ttl)
	q.renew("a")

	expired := q.purge()

	if len(expired) != 1 || expired[0] != "b" {
		t.Fatalf("expected purge to reclaim only [b], got %v", expired)
	}

	// "a" must still be queued (its lease was renewed).
	if len(q.items) != 1 || q.items[0].key != "a" {
		t.Fatalf("expected [a] to remain queued, got %v", q.items)
	}
}

func TestMap_StopIsIdempotent(t *testing.T) {
	m := New[string, int](time.Minute)

	// Must not panic, even when called more than once.
	m.Stop()
	m.Stop()
}
