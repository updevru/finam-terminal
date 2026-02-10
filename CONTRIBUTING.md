# Contributing to Finam Terminal

Thank you for your interest in contributing! We value clear, concise, and high-quality code.

## 🛠 Development Setup

1. **Go Version**: Ensure you have Go 1.24+.
2. **Dependencies**: Run `go mod tidy` to download modules.
3. **Linting**: We use [golangci-lint](https://golangci-lint.run/). To install the correct version, run `go install tool`. You can then run it with `go tool golangci-lint run`.

## 🚀 Workflow

1. **Fork & Clone** the repository.
2. **Create a Branch**: Use descriptive names like `feature/new-widget` or `fix/login-bug`.
3. **Make Changes**:
   - Keep changes atomic and focused.
   - **Documentation**: For significant changes or new features, update or create the corresponding track in the `conductor/` directory (including `spec.md` and `plan.md`) using [Conductor](https://github.com/gemini-cli-extensions/conductor). This helps maintain the project's architectural state.
   - Update code comments if logic changes.
4. **Verify**:
   - Run tests: `go test ./...`
   - Run linter: `go tool golangci-lint run`
5. **Push & PR**: Open a Pull Request against the `main` branch.

## 📝 Code Standards

- **Formatting**: All code must be formatted with `gofmt`.
- **Style**: Follow standard Go idioms (Effective Go).
- **UI/TUI Specifics**:
  - This project uses [`rivo/tview`](https://github.com/rivo/tview).
  - **Critical**: Any UI updates from goroutines (like API callbacks) MUST use `app.QueueUpdateDraw` to ensure thread safety.
- **Configuration**: Never hardcode credentials. Always use the `config` package.

## 💌 Commit Messages

Please follow the [Conventional Commits](https://www.conventionalcommits.org/) format:

- `feat: add position filtering`
- `fix: resolve nil pointer in api client`
- `docs: update readme badges`
- `refactor: simplify render logic`

## ✅ Pull Request Checklist

Before submitting your PR, please ensure:
- [ ] Code compiles without errors.
- [ ] All tests pass (`go test ./...`).
- [ ] Linter checks pass (`go tool golangci-lint run`).
- [ ] You have added tests for new functionality (if applicable).

## ⚖️ License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
