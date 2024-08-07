package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

const (
	exitSuccess int = iota
	exitFailure
)

type config struct {
	port    int
	env     string
	useOsFs bool
}

type application struct {
	config *config
	logger *slog.Logger
}

func newConfig() *config {
	cfg := &config{}
	flag.IntVar(&cfg.port, "port", 4000, "WEBAPP server port")
	flag.StringVar(
		&cfg.env,
		"env",
		"development",
		"Environment (development|staging|production)",
	)
	flag.BoolVar(&cfg.useOsFs, "useOsFs", false, "Choose between embed fs or live fs")
	flag.Parse()
	return cfg
}

func newApplication(cfg *config, logger *slog.Logger) *application {
	return &application{cfg, logger}
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func newServer(port int, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

func main() {
	cfg := newConfig()
	logger := newLogger()
	app := newApplication(cfg, logger)
	srv := newServer(cfg.port, app.routes())

	logger.Info("server started", "env", cfg.env, "address", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("closing server...", "error", err.Error())
		os.Exit(exitFailure)
	}
}
