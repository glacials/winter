# Repository Guidelines

## Project Structure & Module Organization
- `cmd/`: Cobra-based CLI commands (`build`, `serve`, `init`, `freeze`, `test`, etc.).
- `document/`: Core build pipeline (config, parsing, templates, images, URIs).
- `cliutils/`: Small CLI helpers (printing, prompts).
- `testdata/` and `document/testdata/`: Input/fixture files for tests.
- `main.go`: Entrypoint wiring version info to the CLI.
- `.github/workflows/`: CI running `go test` and a release build.

## Build, Test, and Development Commands
- `go test ./...`: Run all unit tests. Add `-race` locally for data race checks.
- `go run . --help`: Preview CLI usage; try commands like `go run . build -s`.
- `go build -o w .`: Compile the CLI locally.
- Versioned build (matches CI): `go build -ldflags="-X 'main.version=$(git describe --tags --always)'" -o w .`.
- Hygiene: `go fmt ./...` and `go vet ./...` before sending PRs.

## Coding Style & Naming Conventions
- Go formatting is canonical: run `go fmt` and keep imports tidy.
- Prefer idiomatic, small packages and clear exported names (CamelCase); files use `snake_case.go`.
- Errors: wrap with context (`fmt.Errorf("doing X: %w", err)`) and log via `slog` where appropriate.
- Keep CLI flags consistent with existing ones (`-v/--verbose`, `-q/--quiet`).

## Testing Guidelines
- Use the standard `testing` package; name files `*_test.go` and tests `TestXxx`.
- Favor table-driven tests; place fixtures under `testdata/`.
- Ensure deterministic outputs (templates, image ops) and cover edge cases.
- Run `go test ./...` (and `-race` locally) before committing.

## Commit & Pull Request Guidelines
- Commits: imperative, concise subject (≤72 chars). Examples: `fix reloader nil channel`, `feat: add build summary`.
- Reference issues/PRs when applicable (`(#123)`). Squash trivial fixups.
- PRs: include a clear description, reproduction or motivation, test coverage for changes, and notes on docs updates (README/CLI help) if behavior changes.

## Architecture Overview
- CLI is composed in `cmd/root.go`; core build logic lives in `document/` (via `Substructure.ExecuteAll`).
- `build -s` serves `dist/` on `localhost:8100` with live reload.
