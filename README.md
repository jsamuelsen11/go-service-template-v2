# Go Service Template V2

![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)
![Architecture](https://img.shields.io/badge/Architecture-Hexagonal-84cc16)
![DI](https://img.shields.io/badge/DI-samber%2Fdo_v2-a855f7)

A production-ready Go service template implementing hexagonal architecture (ports & adapters). Designed as a starting point for building services that integrate with external APIs, with clean separation of concerns and testable layers.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Project Structure](#project-structure)
- [Development](#development)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Architecture](#architecture)
- [Documentation](#documentation)

## Prerequisites

### Go (via mise)

We recommend installing Go through [mise](https://mise.jdx.dev/) for consistent toolchain management:

```bash
# Install mise (if not already installed)
curl https://mise.run | sh

# Install Go (version defined in .mise.toml)
mise install
```

### Task

This project uses [Task](https://taskfile.dev/) as its task runner:

```bash
# Install via mise
mise use -g task

# Or install directly
go install github.com/go-task/task/v3/cmd/task@latest
```

## Quick Start

```bash
# Clone the template
git clone <repo-url> my-service
cd my-service

# Install dependencies
mise install
go mod download

# Run the service
task run

# Run tests
task test
```

## Project Structure

```
internal/
  domain/          # Business entities and rules (zero dependencies)
  ports/           # Interface definitions (services.go, clients.go, health.go)
  app/             # Application services (orchestration, no business logic)
  adapters/
    http/          # Inbound HTTP handlers
    clients/       # Outbound HTTP clients
      acl/         # Anti-Corruption Layer (translation + error mapping)
  platform/        # Cross-cutting concerns (logging, config, middleware)
```

## Development

### Common Tasks

```bash
task run          # Run the service
task test         # Run all tests
task lint         # Run linters
task build        # Build binary
task generate     # Run code generation
task --list       # Show all available tasks
```

### Testing

```bash
task test                  # All tests
task test -- ./internal/domain/...   # Domain tests only
task test -- -run TestName           # Specific test
```

## Configuration

<!-- TODO: Document configuration options after Go code scaffolding -->

Configuration is loaded from environment variables and config files. See `internal/platform/config/` for details.

## API Reference

This service integrates with the TODO API defined in [`todo-service-openapi.yaml`](./todo-service-openapi.yaml).

Endpoints:

| Method | Path                | Description     |
|--------|---------------------|-----------------|
| GET    | `/api/v1/todos`     | List all TODOs  |
| POST   | `/api/v1/todos`     | Create a TODO   |
| GET    | `/api/v1/todos/{id}`| Get a TODO      |
| PUT    | `/api/v1/todos/{id}`| Update a TODO   |
| DELETE | `/api/v1/todos/{id}`| Delete a TODO   |

## Architecture

This template follows **hexagonal architecture** (ports & adapters), keeping business logic isolated from infrastructure concerns. Dependencies always point inward -- adapters depend on ports, never the reverse.

For the full architecture guide with diagrams, layer details, and implementation patterns, see [ARCHITECTURE.md](./docs/ARCHITECTURE.md).

## Documentation

```
docs/
  ARCHITECTURE.md                     # Architecture guide with diagrams and patterns
  adr/
    README.md                         # ADR index
    0001-hexagonal-architecture.md    # Why hexagonal architecture
    template.md                       # Template for new ADRs
```
