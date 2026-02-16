# CLAUDE.md

Instructions for AI agents working on this project.

## Architecture

This project implements **Hexagonal Architecture** (Ports & Adapters) as defined in:

- [ARCHITECTURE.md](./docs/ARCHITECTURE.md) -- comprehensive implementation guide
- [ADR-0001](./docs/adr/0001-hexagonal-architecture.md) -- architectural decision and rationale

### Layer Rules

| Layer           | Location             | May Depend On           | Must NOT Depend On                  |
| --------------- | -------------------- | ----------------------- | ----------------------------------- |
| **Domain**      | `internal/domain/`   | Nothing                 | Ports, App, Adapters, Platform      |
| **Ports**       | `internal/ports/`    | Domain                  | App, Adapters, Platform             |
| **Application** | `internal/app/`      | Domain, Ports           | Adapters, Platform (except logging) |
| **Adapters**    | `internal/adapters/` | Domain, Ports, Platform | App (directly)                      |
| **Platform**    | `internal/platform/` | Nothing (cross-cutting) | Domain, Ports, App, Adapters        |

**Dependency direction**: Always inward. Adapters depend on Ports, never the reverse.

### Key Constraints

- Domain layer has **zero infrastructure dependencies** -- no I/O, no logging, no external packages
- Ports define interfaces; concrete implementations live in Adapters
- Application services orchestrate use cases but contain **no business logic**
- All external API integration goes through the **Anti-Corruption Layer** (`adapters/clients/acl/`)
- ACL uses domain subpackages: `acl/todo/`, `acl/project/` with shared `acl/errors.go`
- Port files are split: `services.go` (service ports), `clients.go` (client ports), `health.go`

## Dependency Injection

Use [`samber/do`](https://github.com/samber/do) v2 for dependency injection:

- Register dependencies with `do.Provide(injector, providerFn)`
- Resolve dependencies with `do.MustInvoke[Type](injector)`
- Use scoped injectors for multi-domain isolation when needed
- No reflection, no code generation -- generics only

## Sub-Domain Strategy

Start with a **single domain + Strategy pattern** for behavioral variations (e.g., work vs personal TODOs):

- Define a `TodoProcessor` interface with category-specific implementations
- The `Todo` entity includes a `Category` field
- Do NOT prematurely split into separate bounded contexts
- Extract to sub-domains only when categories require different entities, APIs, or team ownership

## Coding Standards

- Go conventions: `gofmt`, `goimports`, effective Go idioms
- Error handling: return domain errors (`ErrNotFound`, `ErrValidation`, etc.), not infrastructure errors
- Context: pass `context.Context` as first parameter to all functions that do I/O
- Logging: use `slog` structured logging, inject logger via constructor
- Testing: domain layer tested without mocks; adapters tested with mocked ports

## Implementation Philosophy

- **Minimal implementation**: only build what's needed for the current task
- **No speculative features**: don't add configurability, flags, or abstractions "just in case"
- **Follow existing patterns**: when adding new code, match the style and patterns already established
- **Prefer editing over creating**: modify existing files rather than creating new ones when possible

## OpenAPI Spec

The downstream TODO API is defined in `todo-service-openapi.yaml`. Use it as the source of
truth for domain entity design, handler endpoints, and DTO shapes.
