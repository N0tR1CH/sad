package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/N0tR1CH/sad/internal/data"
	"github.com/N0tR1CH/sad/internal/services"
	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
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
	db      struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
}

type application struct {
	config         *config
	logger         *slog.Logger
	models         data.Models
	services       services.Services
	sessionManager *scs.SessionManager
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
	flag.BoolVar(
		&cfg.useOsFs,
		"useOsFs",
		false,
		"Choose between embed fs or live fs",
	)
	flag.StringVar(
		&cfg.db.dsn,
		"db-dsn",
		"postgres://postgres:postgrespwd@localhost/sad_dev?sslmode=disable",
		"PostgreSQL DSN",
	)
	flag.IntVar(
		&cfg.db.maxOpenConns,
		"db-max-open-conns",
		25,
		"PostgreSQL max open connections",
	)
	flag.IntVar(
		&cfg.db.maxIdleConns,
		"db-max-idle-conns",
		25,
		"PostgreSQL max idle connections",
	)
	flag.StringVar(
		&cfg.db.maxIdleTime,
		"db-max-idle-time",
		"15m",
		"PostgreSQL max connection idle time",
	)
	flag.Parse()

	return cfg
}

func newApplication(
	cfg *config,
	logger *slog.Logger,
	models data.Models,
	services services.Services,
	sessionManager *scs.SessionManager,
) *application {
	return &application{cfg, logger, models, services, sessionManager}
}

func newLogger() *slog.Logger {
	return slog.New(
		log.NewWithOptions(
			os.Stdout,
			log.Options{
				ReportCaller:    true,
				ReportTimestamp: true,
				TimeFormat:      time.DateTime,
				Prefix:          "SAD 😥",
			},
		),
	)
}

func newServer(port int, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		TLSConfig: &tls.Config{
			CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
			MinVersion:       tls.VersionTLS12,
			MaxVersion:       tls.VersionTLS13,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},
	}
}

func newSessionManager(pool *pgxpool.Pool) *scs.SessionManager {
	sm := scs.New()
	sm.Store = pgxstore.New(pool)
	sm.Lifetime = 12 * time.Hour
	return sm
}

func main() {
	cfg := newConfig()
	logger := newLogger()

	db, err := openDB(cfg)
	if err != nil {
		logger.Error("database problem", "err", err)
		os.Exit(exitFailure)
	}
	defer db.Close()
	logger.Info(
		"database connection opened",
		"dbStats",
		fmt.Sprintf("%+v", db.Stats()),
	)

	pool, err := pgxpool.New(context.Background(), cfg.db.dsn)
	if err != nil {
		logger.Error("database problem", "err", err)
		os.Exit(exitFailure)
	}
	defer pool.Close()

	app := newApplication(
		cfg,
		logger,
		data.NewModels(db),
		services.NewServices(logger),
		newSessionManager(pool),
	)
	srv := newServer(cfg.port, app.routes())

	logger.Info("server started", "env", cfg.env, "address", srv.Addr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		if err := srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem"); err != nil &&
			err != http.ErrServerClosed {

			logger.Error("closing server...", "error", err.Error())
			os.Exit(exitFailure)
		}
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	logger.Info("Server Timeout", "Info", "Killing server in 10 seconds")
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Error when server shutdown", "Error", err.Error())
	}
}

func openDB(cfg *config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
