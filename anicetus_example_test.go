package anicetus_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/rafaeljusto/anicetus/v2"
	"github.com/rafaeljusto/anicetus/v2/detector"
	"github.com/rafaeljusto/anicetus/v2/storage"
)

// Request represents an incoming request.
type Request struct {
	Input string
}

// Fingerprint returns the fingerprint that can group many requests together.
func (r Request) Fingerprint() anicetus.Fingerprint {
	hash := sha256.New()
	if len(r.Input) > 7 {
		hash.Write([]byte(r.Input[:7]))
	} else {
		hash.Write([]byte(r.Input))
	}
	return anicetus.Fingerprint(hex.EncodeToString(hash.Sum(nil)))
}

func ExampleAnicetus_Evaluate() {
	ctx := context.Background()

	detector := detector.NewTokenBucketInMemory(
		detector.TokenBucketWithLimitersBurst(1),
		detector.TokenBucketWithLimitersInterval(time.Minute),
		detector.TokenBucketWithCoolDownInterval(10*time.Minute),
	)

	gatekeeperStorage := storage.NewInMemory()

	anicetus := anicetus.NewAnicetus[Request](detector, gatekeeperStorage)

	evaluate := func(req Request) {
		status, err := anicetus.Evaluate(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to evaluate request: %v", err)
		}
		fmt.Printf("status: %v\n", status)
	}

	req := Request{Input: "hello"}

	evaluate(req)
	evaluate(req)
	evaluate(req)

	if err := anicetus.RequestDone(ctx, req); err != nil {
		fmt.Fprintf(os.Stderr, "failed to mark request as done: %v", err)
	}

	evaluate(req)

	// Output:
	// status: open-gates
	// status: process
	// status: wait
	// status: open-gates
}
