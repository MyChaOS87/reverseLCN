# reverseLCN Agent Guide

`reverseLCN` is a specialized, lightweight Go utility suite that bridges and decodes proprietary Local Control Network (LCN) home automation bus traffic over USB serial interfaces (LCN-PKU) and exposes it to MQTT networks.

The repository provides two core CLI binaries:
- [`cmd/lcn2mqtt`](./cmd/lcn2mqtt/lcn2mqtt.go): A bidirectional USB serial <-> MQTT bridge that deserializes bus packets into JSON topics and transmits outbound MQTT commands onto the physical LCN bus.
- [`cmd/lcnMonitor`](./cmd/lcnMonitor/lcnMonitor.go): A dedicated traffic monitoring subscriber that ingests bus telemetry from MQTT and prints human-readable interpretations of relay states, status queries, and device commands.

---

## Engineering Principles

- **Pre-v1 Engineering Integrity**: `reverseLCN` is currently pre-v1 software. Never introduce backwards-compatibility shims, deprecated legacy wrappers, or speculative abstraction layers. Refactor callers in-place and remove superseded code cleanly.
- **80/20 Simplicity & Protocol Focus**: Favor straightforward, idiomatic Go standard library code over heavy external frameworks. Keep package APIs lean, deterministic, and tailored directly to physical bus constraints (fixed frame lengths, bitwise math, serial buffer boundaries).
- **Strict Boundary Decoupling**: Generic packages under [`pkg/`](./pkg) must remain completely independent of application concerns. They must never depend on [`cmd/`](./cmd), [`config/`](./config), or [`internal/`](./internal) (enforced by `depguard`).

---

## Knowledgebase Routing Table

Consult these dedicated documents for detailed technical specifications and patterns:

| Document | Scope & When to Read |
|---|---|
| [`CONTEXT.md`](./CONTEXT.md) | **Domain Glossary & Ubiquitous Language**: Definitions of `LCN-PKU`, physical packet frames (6/8/12/20 bytes), bit-mirrored [`Src`](./internal/serial/chunker/lcn/lcn.go#L35), `Info` length mask (`0x0C`), rotated-sum [`Checksum`](./internal/serial/chunker/lcn/lcn.go#L160), [`packet.Packet`](./pkg/serial/chunker/packet/packet.go#L10), [`chunker.Chunker`](./pkg/serial/chunker/chunker.go#L19), [`serial.Port`](./pkg/serial/port.go#L18), [`broker.Broker`](./pkg/broker/broker.go#L5), and [`monitor.DataStore`](./internal/monitor/datastore.go#L14). |
| [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) | **System Architecture**: Dual-binary architecture, package DAG (`pkg/` vs `internal/` boundary), bidirectional bridge sequence diagram, sliding-window chunker resynchronization state machine, and multi-stage `FROM scratch` container builds. |
| [`docs/CONVENTIONS.md`](./docs/CONVENTIONS.md) | **Coding Idioms & Standards**: Go 1.27 conventions, bitwise mask constants, sentinel error wrapping (`%w`), functional options for serial and MQTT brokers, Viper environment mapping (`LCN_*`), and commit conventions. |
| [`docs/TESTING.md`](./docs/TESTING.md) | **Test Philosophy & Compliance**: Output-oriented tabular testing for binary protocols, 3-tier testing model, `t.Parallel()` and `t.Context()` compliance, test assertion ordering, and the 4 tracked backlog items (#13-#16). |
| [`docs/TOOLCHAIN.md`](./docs/TOOLCHAIN.md) | **Developer Toolchain**: Complete [`Makefile`](./Makefile) target reference, `export GOWORK=off`, isolated tool modules with `dummy.go`, `go fix`, vulnerability scanning (`syft`, `grype`, `govulncheck`), and GPL-3.0 / EPL-2.0 licensing policies. |

---

## Git & Solo Linear-`main` Workflow

- **Linear-`main` Development**: Solo maintainer mode is active. Work directly on `main` for cohesive changes, or create an isolated feature branch / worktree and merge back to `main` using fast-forward only (`git merge --ff-only`).
- **Remote Mutation Safeguard**: Never run `git push`, `git push --force`, or any command that mutates git remotes without explicit, affirmative user confirmation.

---

## Sandbox Execution Matrix (`BypassSandbox`)

Prefer running commands in the default sandboxed environment (`BypassSandbox: false`). Escalate to `BypassSandbox: true` only when encountering genuine environment boundaries:

| Operation | `BypassSandbox` | Reason |
|---|---|---|
| `make test`, `make coverage`, `go test ./...` | `false` | Runs entirely in-memory using local workspace mounts and stdlib tests. |
| `make lint`, `make build`, `make tidy`, `make sbom`, `make notice` | `false` | Offline static analysis, AST verification, and local file generation using pre-installed toolchains. |
| `make audit-image`, `make docker-build` | `true` | Requires Podman or Docker container engine socket access (`/run/user/...` or `/var/run/docker.sock`). |
| `make audit-repo`, `make vulncheck`, `make audit` | `true` | Requires outbound network access to fetch remote vulnerability databases (Grype DB and Go vulnerability database). |
| `make upgrade-direct-dependencies`, `make tools-update`, `git fetch/pull/push` | `true` | Requires outbound network access to Go module proxies and remote git repositories. |

---

## Agent Hazards (Do Not Do)

1. **Never allow `pkg/` to import `cmd/`, `config/`, or `internal/`**: The `pkg/` tree provides reusable, generic primitives (serial I/O, MQTT abstraction, chunking). Enforced strictly by `.golangci.yml` `depguard` rules.
2. **Never alter `mirrorSrc` or `calcChecksum` without verified bus vectors**: LCN hardware bit-reversal and rotated-sum checksums are reverse-engineered algorithms matching physical Issendorff hardware. Changing them without captured real-bus test vectors breaks communication.
3. **Never block the MQTT callback goroutine**: The Paho MQTT callback dispatches inbound messages. Long-running or blocking work (such as sending to serial) must be dispatched asynchronously (e.g., `go port.Send(buf)`) to prevent stalling the MQTT message loop.
4. **Never hand-edit generated license or SBOM assets**: Do not manually edit [`NOTICE.txt`](./NOTICE.txt), `dependencies.asciidoc`, or `reverseLCN.sbom.cdx.json`. Re-generate them via `make notice` and `make sbom`.
5. **Never delete or strip `tools/<tool>/dummy.go` files**: Each isolated tool module requires a `dummy.go` file with a blank import `import _ "<pkg>"` to pin direct tool versions and prevent `go mod tidy` from stripping tool dependencies.
6. **Never remediate transitive CVEs by promoting indirect dependencies**: Do not convert an `// indirect` dependency into a direct dependency in [`go.mod`](./go.mod). Use a version-pinned `replace` directive instead:
   ```
   replace <module> <vulnerable-version> => <module> <fixed-version> // fix <CVE-ID>
   ```
