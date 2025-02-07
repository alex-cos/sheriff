# Sheriff

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-blue)](https://go.dev/)
[![Test Status](https://github.com/alex-cos/sheriff/actions/workflows/test.yml/badge.svg)](https://github.com/alex-cos/sheriff/actions/workflows/test.yml)
[![Lint Status](https://github.com/alex-cos/sheriff/actions/workflows/lint.yml/badge.svg)](https://github.com/alex-cos/sheriff/actions/workflows/lint.yml)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/alex-cos/sheriff)](https://goreportcard.com/report/github.com/alex-cos/sheriff)

**Sheriff** is a lightweight CLI tool to monitor and manage multiple processes with dependency ordering.

It reads a YAML configuration file, starts services in dependency order (dependencies first), logs a periodic JSON status report, supports automatic restart of failed services, and shuts everything down gracefully on `SIGINT`/`SIGTERM`.

## Usage

### 1. Create a config file

```yaml
# config.yaml
stopTimeout: 30s
logLevel: info

monitor:
  period: 60s
  restart: true

services:
  - name: database
    command: "postgres"
    arguments: ["-D", "/var/lib/postgresql/data"]

  - name: api
    command: "server"
    arguments: ["--verbose"]
    dependsOn: ["database"]

  - name: frontend
    command: "nginx"
    arguments: ["-g", "daemon off;"]
    dependsOn: ["api"]
    maxRetries: 5
    restartDeps: true
```

### 2. Run

```bash
sheriff -c config.yaml
```

The tool starts all services in dependency order, prints a JSON status report every `period` seconds, and restarts any service that stops (if `restart: true`). Press `Ctrl+C` to stop all services gracefully.

### 3. Output example

```log
{"level":"INFO","msg":"monitor is starting"}
{"level":"INFO","msg":"service has started successfully","name":"database"}
{"level":"INFO","msg":"service has started successfully","name":"api"}
{"level":"INFO","msg":"status","root":"All running"}
{"level":"WARN","msg":"service is not running, restarting","name":"api"}
{"level":"INFO","msg":"shutting down"}
{"level":"INFO","msg":"shutdown complete"}
```

## Build

```bash
make build
```

The binary is placed in `./bin/`.

## Commands

```bash
make test        # run tests
make lint        # run golangci-lint (output to tmp/lintreport.xml)
make test-cover  # run tests with coverage report
make clean       # clean build artifacts
```

## Configuration reference

### Global options

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `stopTimeout` | duration | `30s` | Max time to wait for a graceful shutdown before force-killing (min 5s) |
| `logLevel` | string | `info` | Log level: `debug`, `info`, `warn`, or `error` |

### Monitor

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `monitor.period` | duration | `60s` | Interval between status reports (min 10s) |
| `monitor.restart` | bool | `false` | Automatically restart services that stop |

### Services

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `name` | string | — | Unique service identifier **(required)** |
| `command` | string | — | Executable path or name **(required)** |
| `arguments` | `[]string` | `[]` | Command-line arguments |
| `dependsOn` | `[]string` | `[]` | Services that must start before this one |
| `maxRetries` | int | `3` | Number of launch attempts before giving up |
| `restartDeps` | bool | `false` | When restarting, also restart all dependencies |

### Notes

- **Dependency graph**: services are started in dependency order (dependencies first) and stopped in reverse. Circular dependencies are detected and rejected at startup.
- **PID reuse**: process identity is verified by comparing creation timestamps, preventing false positives from recycled PIDs.
- **Thread safety**: internal data structures are protected by mutexes for safe concurrent access (monitor goroutine + signal handler).
- **Configuration validation**: the config file is validated on load (required fields, duplicate names, invalid log levels, etc.).
