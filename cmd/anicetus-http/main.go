// Package main is a microservice to control thundering herd problem. The
// service aims to identify similar requests and group them together to reduce
// the load on the backend services.
package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	anicetushttp "github.com/rafaeljusto/anicetus/internal/http"
)

func main() {
	defer handleExit()

	config, errs := anicetushttp.ParseFromEnvs()
	if errs != nil {
		// We are using a logger to print the errors because we don't have a
		// logger yet. We could use the standard logger, but it's better to
		// create a logger with the correct configuration.
		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))
		for _, err := range multierr(errs) {
			logger.Error("failed to parse configuration",
				slog.String("error", err.Error()),
			)
		}
		exit(exitCodeInvalidInput)
	}

	resources := anicetushttp.NewResources(config)

	listener, err := net.Listen("tcp", ":"+strconv.FormatInt(config.Port, 10))
	if err != nil {
		resources.Logger.Error("failed to listen",
			slog.String("error", err.Error()),
		)
		exit(exitCodeSetupFailure)
	}

	resources.Logger.Info("starting web server",
		slog.String("address", listener.Addr().String()),
	)

	router := http.NewServeMux()
	anicetushttp.RegisterHandlers(router, config, resources)
	server := http.Server{
		Handler: router,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Serve(listener); err != nil {
			if err != http.ErrServerClosed {
				resources.Logger.Error("failed to serve",
					slog.String("error", err.Error()),
				)
			}
		}
	}()

	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	if err := server.Shutdown(ctx); err != nil {
		resources.Logger.Error("server shutdown failed",
			slog.String("error", err.Error()),
		)
	}
	resources.Logger.Info("server stopped")
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

// multierr unwraps multiple errors from a single error.
//
// https://pkg.go.dev/errors#Join
func multierr(errs error) []error {
	if errs == nil {
		return nil
	}
	if multierr, ok := errs.(interface{ Unwrap() []error }); ok {
		return multierr.Unwrap()
	}
	return []error{errs}
}
