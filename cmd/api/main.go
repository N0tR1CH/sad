package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/gob"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/N0tR1CH/sad/internal/data"
	"github.com/N0tR1CH/sad/internal/mailer"
	"github.com/N0tR1CH/sad/internal/services"
	"github.com/N0tR1CH/sad/views/components"
	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func init() {
	gob.Register(components.AlertProps{})
}

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
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

type application struct {
	config         *config
	logger         *slog.Logger
	models         data.Models
	permissions    []data.Permission
	services       services.Services
	sessionManager *scs.SessionManager
	mailer         mailer.Mailer
	wg             sync.WaitGroup
}

func newConfig(logger *slog.Logger) *config {
	cfg := &config{}

	// Port on which server starts
	flag.IntVar(&cfg.port, "port", 4000, "WEBAPP server port")

	// Environment type
	//
	// Cors configuration depend on it
	flag.StringVar(
		&cfg.env,
		"env",
		"development",
		"Environment (development|staging|production)",
	)

	// Filesystem type
	//
	// Static files depend on it
	flag.BoolVar(
		&cfg.useOsFs,
		"useOsFs",
		false,
		"Choose between embed fs or live fs",
	)

	// Database configuration
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

	// Mailer configuration
	flag.StringVar(
		&cfg.smtp.host,
		"smtp-host",
		"127.0.0.1",
		`Mail server host:
			- For mailcrab it is localhost -> 127.0.0.1`,
	)

	flag.IntVar(
		&cfg.smtp.port,
		"smtp-port",
		1025,
		`Mail server port:
			- For mailcrab it is 1025`,
	)

	flag.StringVar(
		&cfg.smtp.username,
		"smtp-username",
		"Sadman",
		`Mail server host:
			- For mailcrab any is accepted`,
	)

	flag.StringVar(
		&cfg.smtp.password,
		"smtp-password",
		"Sadman",
		`Mail server host:
			- For mailcrab any is accepted`,
	)

	flag.StringVar(
		&cfg.smtp.sender,
		"smtp-sender",
		"Sad@sad.dev",
		`Mail server sender:
			- For mailcrab any is accepted`,
	)

	flag.Parse()

	logger.Info(
		"config values initialized",
		"smtp-cfg", fmt.Sprintf("%+v", cfg.smtp),
	)

	return cfg
}

func newApplication(
	cfg *config,
	logger *slog.Logger,
	models data.Models,
	services services.Services,
	sessionManager *scs.SessionManager,
	mailer mailer.Mailer,
) *application {
	return &application{
		config:         cfg,
		logger:         logger,
		models:         models,
		services:       services,
		sessionManager: sessionManager,
		mailer:         mailer,
		wg:             sync.WaitGroup{},
	}
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

func newServer(port int, env string, handler http.Handler) *http.Server {
	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	if env == "development" {
		tlsCfg := &tls.Config{
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
		}
		s.TLSConfig = tlsCfg
	}
	return s
}

func newSessionManager(pool *pgxpool.Pool) *scs.SessionManager {
	sm := scs.New()
	sm.Store = pgxstore.New(pool)
	sm.Lifetime = 12 * time.Hour
	return sm
}

func main() {
	logger := newLogger()
	cfg := newConfig(logger)

	// Database connection
	db, err := openDB(cfg)
	if err != nil {
		logger.Error("database problem", "err", err)
		os.Exit(exitFailure)
	}
	defer func() {
		_ = db.Close()
	}()
	logger.Info(
		"database connection opened",
		"dbStats",
		fmt.Sprintf("%+v", db.Stats()),
	)

	// Database connection for session manager
	pool, err := pgxpool.New(context.Background(), cfg.db.dsn)
	if err != nil {
		logger.Error("database problem", "err", err)
		os.Exit(exitFailure)
	}
	defer pool.Close()
	logger.Info(
		"database connection for session manager opened",
		"dbStats",
		fmt.Sprintf("%+v", pool.Stat()),
	)

	newApplication(
		cfg,
		logger,
		data.NewModels(db, logger),
		services.NewServices(logger),
		newSessionManager(pool),
		mailer.New(
			cfg.smtp.host,
			cfg.smtp.port,
			cfg.smtp.username,
			cfg.smtp.password,
			cfg.smtp.sender,
			cfg.env,
		),
	).serve()
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

func (app *application) serve() {
	srv := newServer(app.config.port, app.config.env, app.routes())
	app.logger.Info(
		"server started",
		"env", app.config.env,
		"address", srv.Addr,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		switch app.config.env {
		case "development":
			if err := srv.ListenAndServeTLS(
				"./tls/cert.pem",
				"./tls/key.pem",
			); err != nil && err != http.ErrServerClosed {
				app.logger.Error("closing server...", "error", err.Error())
				os.Exit(exitFailure)
			}
		case "staging":
			fallthrough
		case "production":
			if err := srv.ListenAndServe(); err != nil {
				app.logger.Error("closing server...", "error", err.Error())
				os.Exit(exitFailure)
			}
		default:
			app.logger.Error("such env type does not exist",
				"env", app.config.env,
			)
			os.Exit(exitFailure)
		}
	}()

	// Wait for interrupt signal to gracefully
	// shutdown the server with a timeout of 10 seconds.
	<-ctx.Done()

	// Wait for background job to be finished
	app.wg.Wait()

	app.logger.Info("Server Timeout", "Info", "Killing server in 10 seconds")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		app.logger.Error("Error when server shutdown", "Error", err.Error())
	}
}
