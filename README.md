# runlit CLI

[![license: MIT](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)
[![go report](https://goreportcard.com/badge/github.com/runlit-dev/cli?style=flat-square)](https://goreportcard.com/report/github.com/runlit-dev/cli)

Evaluate AI-generated code from your terminal.

## Install

```bash
# macOS / Linux (Homebrew)
brew install runlit-dev/tap/runlit

# Windows (winget)
winget install runlit.cli

# Go
go install github.com/runlit-dev/cli@latest
```

## Usage

```bash
# Evaluate a pull request
runlit check --pr 42 --repo org/repo

# Evaluate a local diff file
runlit eval --diff ./changes.diff

# Set your API key
runlit auth login

# Show current config
runlit config show
```

## Output

```
runlit eval — BLOCK 🔴
Score: 41/100

Signal       Score   Notes
──────────────────────────────────────────────────────────
Hallucination  0.42  stripe.PaymentMethod.attach_async — method does not exist
Intent         1.00  —
Security       1.00  No issues found
Compliance     1.00  No violations

Eval ID: 01jxyz...  Latency: 312ms
```

## Configuration

```yaml
# .runlit.yml (project root)
thresholds:
  block: 50
  warn: 70
signals:
  hallucination: true
  intent: true
  security: true
  compliance: false
```

## Stack

- `cobra` — CLI framework
- `github.com/google/uuid` — UUIDv7 correlation IDs
- Calls `api.runlit.dev/v1/eval` — uses shared Go types from `core/contracts`

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Issues and PRs welcome.

## License

MIT — see [LICENSE](LICENSE).
