# Local Checks

This project has optional local Git hooks that run the same checks as the CI
pipeline before code leaves your machine.

## Enable Hooks

Run this once per clone:

```bash
git config core.hooksPath .githooks
```

This is intentionally not changed automatically because `core.hooksPath` is a
local repository setting.

## Hook Behavior

`pre-commit` runs the fast local gate:

```bash
make check-local
```

It checks:

- `gofmt` status for tracked Go files in `backend` and `backup-service`.
- `git diff --check`.
- GitHub Actions workflow YAML parsing.
- Shell syntax for deploy scripts.
- Backend `errcheck` through `golangci-lint` for staged backend Go packages.

`pre-push` runs the full local CI gate:

```bash
make check-push
```

It checks:

- Everything from `check-local`.
- Deploy compose rendering with minimal test environment values.
- Backend tests: `cd backend && go test ./...`.
- Backup service tests: `cd backup-service && go test ./...`.
- Backend lint: `cd backend && golangci-lint run ./...`.
- Frontend install, lint, and build: `cd frontend && npm ci && npm run lint && npm run build`.

## Manual Commands

Fast local checks:

```bash
make check-local
```

Full local push checks:

```bash
make check-push
```

Production-oriented release checks:

```bash
make check-release
```

`check-release` includes `check-push` and local Docker builds for backend,
frontend, and backup-service images.

Optional end-to-end checks:

```bash
make check-e2e
```

This starts the local Docker Compose stack and runs Playwright tests. It is not
part of `pre-push` because it is slower and requires a working local Docker
environment.

## Local Dependencies

Install these locally:

- Go 1.26.x.
- A Go toolchain capable of running the pinned `golangci-lint` from `Makefile`.
- Node.js 20 and npm.
- Docker Engine with Docker Compose plugin.
- Ruby, used only to parse `.github/workflows/ci-cd.yml`.
- Optional: Playwright browser dependencies for `make check-e2e`.

## Known Limitations

Local hooks catch code, formatting, lint, build, workflow syntax, and compose
configuration errors. They cannot guarantee that remote deployment succeeds,
because deploy jobs also depend on GitHub Environment secrets, GHCR permissions,
SSH access, server health, DNS, and TLS setup.

`Makefile` runs a pinned `golangci-lint` through `go run` by default, so an old
globally installed `golangci-lint` binary is not used. To use a local binary
instead, override `GOLANGCI_LINT`:

```bash
GOLANGCI_LINT=golangci-lint make lint
```

The focused errcheck target runs only for staged backend Go packages:

```bash
make check-backend-errcheck
```

For emergencies, Git allows bypassing hooks:

```bash
git push --no-verify
```

Use this only when the failure is understood and CI is expected to remain green.
