# Contributing to Finam Terminal

Thank you for your interest in contributing! We value clear, concise, and high-quality code.

## 🛠 Development Setup

1. **Go Version**: Ensure you have Go 1.26+.
2. **Dependencies**: Run `go mod tidy` to download modules.
3. **Linting**: We use [golangci-lint](https://golangci-lint.run/). Please install it locally.
4. **Make** (optional but recommended): a `Makefile` is provided with shortcuts for the common dev loop — `make build`, `make test`, `make test-integration`, `make test-all`, `make test-race`, `make coverage`, `make lint`. `make build` injects version metadata via `-ldflags` from `git describe`.

## 🚀 Workflow

1. **Fork & Clone** the repository.
2. **Create a Branch**: Use descriptive names like `feature/new-widget` or `fix/login-bug`.
3. **Make Changes**:
   - Keep changes atomic and focused.
   - **Documentation**: For significant changes or new features, update or create the corresponding track in the `conductor/` directory (including `spec.md` and `plan.md`) using [Conductor](https://github.com/gemini-cli-extensions/conductor). This helps maintain the project's architectural state.
   - Update code comments if logic changes.
4. **Verify**:
   - Run unit tests: `make test` (or `go test ./...`)
   - Run integration tests: `make test-integration` (or `go test -tags=integration ./api/...`). These hit an in-process mock gRPC server (`api/testserver/`, built on `bufconn`) — no token, no network.
   - Run with race detector: `make test-race` (requires `CGO_ENABLED=1`)
   - Run linter: `make lint` (or `golangci-lint run`)
5. **Push & PR**: Open a Pull Request against the `main` branch.

## 📝 Code Standards

- **Formatting**: All code must be formatted with `gofmt`.
- **Style**: Follow standard Go idioms (Effective Go).
- **UI/TUI Specifics**:
  - This project uses [`rivo/tview`](https://github.com/rivo/tview).
  - **Critical**: Any UI updates from goroutines (like API callbacks) MUST use `app.QueueUpdateDraw` to ensure thread safety.
- **Configuration**: Never hardcode credentials. Always use the `config` package.
- **gRPC API additions**: When you add a new method to `api.Client`, also wire a corresponding mock entry into `api/testserver/` and add an integration test in `api/client_*_integration_test.go`. This keeps the integration suite a faithful reflection of the real client surface.

## 💌 Commit Messages

Please follow the [Conventional Commits](https://www.conventionalcommits.org/) format:

- `feat: add position filtering`
- `fix: resolve nil pointer in api client`
- `docs: update readme badges`
- `refactor: simplify render logic`

## ✅ Pull Request Checklist

Before submitting your PR, please ensure:
- [ ] Code compiles without errors.
- [ ] Unit tests pass (`make test` / `go test ./...`).
- [ ] Integration tests pass (`make test-integration` / `go test -tags=integration ./api/...`).
- [ ] Linter checks pass (`make lint` / `golangci-lint run`).
- [ ] You have added tests for new functionality (if applicable).
- [ ] Coverage of `api/client.go` has not regressed (verify with `make coverage`).

## ⚖️ License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
