# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go CLI application for intelligent household chore scheduling. Uses SQLite for storage and Cobra for CLI commands. This repository contains comprehensive design documents; the Go source code implementation follows the specifications in `/docs/`.

## Build Commands

```bash
make build                 # Build binary to bin/chore-scheduler
make test                  # Run all tests with coverage
make run                   # Run with go run
go test -v ./internal/...  # Run specific package tests
go test -v -run TestName ./internal/scheduler/  # Run single test
```

Cross-compilation targets: `make build-linux-amd64`, `make build-linux-arm64`

## Architecture

Layered architecture with clear separation:

```
cmd/chore-scheduler/main.go    # Entry point
internal/
├── cli/          # Cobra commands and handlers
├── db/           # SQLite connection, migrations
├── models/       # Task, Completion, ScheduledTask, Config structs
├── repository/   # Data access layer (repository pattern)
└── scheduler/    # Core scheduling algorithm and priority calculation
```

**Data flow**: CLI → Repository → DB. The scheduler calculates task priorities and assigns tasks to days based on effort capacity.

## Database

SQLite with tables: `tasks`, `completions`, `scheduled_tasks`, `config`. Default location: `~/.chore-scheduler/chore.db` (override with `--db` flag or `CHORE_SCHEDULER_DB` env var).

## Key Concepts

- **Effort levels**: 1 (quick, 5-15 min), 2 (medium, 15-45 min), 3 (long, 45+ min)
- **max_daily_effort**: Config value limiting total effort per day (default: 10)
- **Priority**: Overdue tasks get priority 1000+, due tasks get 1000, future tasks get 100/daysUntil

## Dependencies

- `github.com/mattn/go-sqlite3` - SQLite driver (requires CGO)
- `github.com/spf13/cobra` - CLI framework
- `github.com/stretchr/testify` - Testing
- `github.com/olekukonko/tablewriter` - Table output
