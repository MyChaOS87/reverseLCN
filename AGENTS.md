# reverseLCN Agent Guide

This document describes reverseLCN-specific conventions and tooling.

---

## Project Overview

`reverseLCN` is a utility application that acts as a bridge between the serial LCN-PKU bus interface and an MQTT broker.

---

## Common Patterns

```bash
make all          # Full CI pipeline (tidy → build → lint → test)
make test         # Run all tests
make lint         # Run golangci-lint via isolated tool modfile
make tidy         # go mod tidy
make github-actions-test # Run GitHub Actions locally using act
make tools-update # Update tool dependencies
make tools-tidy   # Tidy tool modules
make upgrade-direct-dependencies # Upgrade direct dependencies
```

**Tool Isolation & Version Pinning**: Tools live in `tools/<tool>/go.mod`, invoked as:
```bash
go tool -modfile=tools/golangci-lint/go.mod golangci-lint run ./...
```

To prevent `go mod tidy` from stripping tool dependencies (or relegating them entirely to `// indirect`), each tool module contains a `dummy.go` file with a blank import of the tool's main package:
```go
package <tool>
import _ "<tool-package-path>"
```
This ensures the tool is tracked as a direct dependency in `go.mod`, pinning its exact version.


**CI Gates**: `tidy` must produce no changes (enforced via `git diff --exit-code`).

**Commit Format**:
```
feat[(<topic>)]: <description>
chore[(<topic>)]: <description>
bugfix[(<topic>)]: <description>
```
- Active present tense, one heading only
- Use `*` for bullet points
- Escape backticks in shell

---

## Architecture & Layout

| Component | Directory | Description |
|-----------|-----------|-------------|
| **CLI binaries** | `cmd/` | Contains `lcn2mqtt` (main bridge) and `lcnMonitor` (monitoring subscriber) |
| **MQTT Client** | `pkg/broker/mqtt` | MQTT broker client interface (Paho MQTT) |
| **Serial Bus** | `pkg/serial` | Serial port reader/writer loop |
| **LCN Protocol** | `internal/serial/chunker/lcn` | Packet deserializer, checksum calculation, and layout mapping |
| **Collector Chunker** | `pkg/serial/chunker` | Buffer chunking logic separating byte streams into discrete packets |

---

## Git Workflow

**Branching**: Prefer feature branches for any non-trivial work:
```bash
git checkout -b feature/description
git checkout -b bugfix/description
```

**Testing Changes**: Always verify changes using the project's make targets before committing:
```bash
make all          # Run full CI pipeline locally
# Or at minimum:
make test && make lint
```

**CRITICAL**: Never push to remote without explicit command. Always verify via `git diff` before pushing.

---

## Troubleshooting

**Run single test**:
```bash
go test -v -run TestName ./package/...
```

**Tool not found?**:
```bash
ls tools/<tool>/          # Check tool exists
make tools-update         # Update dependencies
make tools-tidy && make tidy   # Clean up modules
```
