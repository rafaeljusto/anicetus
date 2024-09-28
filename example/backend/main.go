// Package main is a backend simulation. It's a simple HTTP server that
// simulates some work. It's used to demonstrate the thundering herd problem.
package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	defer handleExit()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	port := os.Getenv("ANICETUS_EXAMPLE_BACKEND_PORT")
	if port == "" {
		logger.Error("missing port",
			slog.String("error", "missing environment variable ANICETUS_EXAMPLE_BACKEND_PORT"),
		)
		exit(exitCodeInvalidInput)
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Error("failed to listen",
			slog.String("error", err.Error()),
		)
		exit(exitCodeSetupFailure)
	}

	logger.Info("starting web server",
		slog.String("address", listener.Addr().String()),
	)

	var numberOfRequests int64
	var cacheEnabled atomic.Bool
	var stats struct {
		requests atomic.Int64
		timeouts atomic.Int64
		success  atomic.Int64
		cached   atomic.Int64
	}
	finishStats := make(chan struct{})

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-finishStats:
				logger.Info("stats",
					slog.Int64("requests", stats.requests.Load()),
					slog.Int64("timeouts", stats.timeouts.Load()),
					slog.Int64("success", stats.success.Load()),
					slog.Int64("cached", stats.cached.Load()),
				)
				return
			case <-ticker.C:
				logger.Info("stats",
					slog.Int64("requests", stats.requests.Load()),
					slog.Int64("timeouts", stats.timeouts.Load()),
					slog.Int64("success", stats.success.Load()),
					slog.Int64("cached", stats.cached.Load()),
				)
				stats.requests.Store(0)
				stats.timeouts.Store(0)
				stats.success.Store(0)
				stats.cached.Store(0)
			}
		}
	}()

	var server http.Server
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&numberOfRequests, 1)
		defer atomic.AddInt64(&numberOfRequests, -1)

		ctx := r.Context()
		stats.requests.Add(1)

		// backend timeout
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if r.Header.Get("Anicetus-Status") == "process" {
			cacheEnabled.Store(true)
			stats.success.Add(1)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if cacheEnabled.Load() {
			stats.cached.Add(1)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		select {
		case <-ctx.Done():
			stats.timeouts.Add(1)
			http.Error(w, "timeout", http.StatusServiceUnavailable)

		// initial retention to wait to the thundering herd to gather
		case <-time.After(time.Second):
			select {
			case <-ctx.Done():
				stats.timeouts.Add(1)
				http.Error(w, "timeout", http.StatusServiceUnavailable)

			// simulate the actual work
			case <-time.After(time.Duration(atomic.LoadInt64(&numberOfRequests)) * time.Second):
				stats.success.Add(1)
				w.WriteHeader(http.StatusNoContent)
			}
		}
		time.Sleep(time.Second)
	})

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Serve(listener); err != nil {
			if err != http.ErrServerClosed {
				logger.Error("failed to serve",
					slog.String("error", err.Error()),
				)
			}
		}
	}()

	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown failed",
			slog.String("error", err.Error()),
		)
	}
	logger.Info("server stopped")

	close(finishStats)
}

type exitCode int

const (
	exitCodeOK exitCode = iota
	exitCodeInvalidInput
	exitCodeSetupFailure
)

type exitData struct {
	code exitCode
}

// exit allows to abort the program while still executing all defer statements.
func exit(code exitCode) {
	panic(exitData{code: code})
}

// handleExit exit code handler.
func handleExit() {
	if e := recover(); e != nil {
		if exit, ok := e.(exitData); ok {
			os.Exit(int(exit.code))
		}
		panic(e)
	}
}
