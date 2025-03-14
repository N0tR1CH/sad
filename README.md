# Share and discuss

Hackernews like web application created in Go.

## Prerequisites

In order to run application in dev mode you may need:

- Go compiler/runtime
- docker with docker compose
- tailwindcss compiler
- overmind task manager
- templ compiler
- pnpm

### Environment file

```bash
❯ tree .env
.env
├── development
│   └── database
└── production
    └── mail
```

```bash
❯ bat .env/development/database
───────┬─────────────────────────────────────────────────────────────────────────────
       │ File: .env/development/database
───────┼─────────────────────────────────────────────────────────────────────────────
   1   │ POSTGRES_USER=postgres
   2   │ POSTGRES_PASSWORD=postgrespwd
   3   │ POSTGRES_DB=sad_dev
   4   │ DSN_STRING=postgres://postgres:postgrespwd@localhost/sad_dev?sslmode=disable
─────────────────────────────────────────────────────────────────────────────────────
```

You might need to generate ssl certificate (required by http 2 standard).

```bash
go run $(go env GOROOT)/src/crypto/tls/generate_cert.go --host localhost

```

Required structure:
```bash
❯ tree tls
tls
├── cert.pem
└── key.pem
```

**dev.sh** file run application.
