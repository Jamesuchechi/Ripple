# Contributing to Ripple

Thank you for your interest in contributing to **Ripple**! We welcome bug reports, feature requests, documentation improvements, and pull requests.

## Development Setup

### Prerequisites
- **Go**: 1.22 or higher
- **Docker & Docker Compose** (for PostgreSQL & Redis)
- **Node.js**: v18+ & npm (for Dashboard and SDKs)

### 1. Clone & Start Infrastructure
```bash
git clone https://github.com/your-org/ripple.git
cd ripple

# Spin up Postgres and Redis containers
docker-compose up -d
```

### 2. Run Database Migrations
```bash
go run ./cmd/migrate
```

### 3. Run Test Suite
Before submitting any pull request, ensure all package test suites pass:
```bash
go test -p 1 -v ./...
```

## Pull Request Guidelines

1. **Keep Commits Clean & Atomic**: Follow standard conventional commits (`feat:`, `fix:`, `docs:`, `test:`).
2. **Never Break API Contracts**: Ensure all changes to `/v1/` endpoints maintain backward compatibility.
3. **Include Unit/Integration Tests**: Any new service, dispatcher, or store logic must include corresponding `_test.go` coverage.
4. **Obey Multi-Tenant Isolation**: Always verify that Redis keys and Postgres queries filter strictly by `project_id`.

## Code Style & Conventions

- Format Go code using `gofmt` or `goimports`.
- Handle idiomatic Go errors explicitly (`if err := rows.Err(); err != nil`).
- Use structured JSON logging via `ripple/internal/logger`.
