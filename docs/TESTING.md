# reverseLCN Test Philosophy & Compliance

This document describes the testing standards, test architecture tiers, and mandatory test mechanics for `reverseLCN`.

---

## Output-Oriented Tabular Testing Philosophy

Protocol bridges and binary serial chunkers must be tested through **output-oriented assertions** rather than mock interaction choreography:

1. **Assert Concrete Outputs**: Supply raw input byte sequences (including partial frames, noise prefixes, concatenated frames, and corrupted bytes) and assert the exact slice of resulting decoded structs or serialized bytes.
2. **Avoid Callback Mock Choreography**: Do not use `testify/mock` to assert that a callback method was called `N` times with specific internal pointers. Instead, supply a lightweight closure that appends received packets to a local slice, and assert against the resulting slice.
3. **Table-Driven Structure**: Organize tests in tabular format:
   ```go
   tests := []struct {
       name     string
       input    []byte
       expected packet.Packet
       err      error
   }{ ... }
   ```

---

## 3-Tier Testing Architecture

```
+----------------------------------------------------------------------+
| Tier 3: Containerized Integration Tests (Optional)                   |
| Real Mosquitto MQTT broker via Podman / Testcontainers               |
+----------------------------------------------------------------------+
| Tier 2: In-Memory Component & Pipeline Fakes                         |
| In-memory pipes (net.Pipe), mqtttest.NewBroker() & Simulator         |
+----------------------------------------------------------------------+
| Tier 1: Pure Protocol & Chunker Tabular Unit Tests                   |
| lcn.Deserialize, lcn.Serialize, Chunker.Collect, monitor.DataStore   |
+----------------------------------------------------------------------+
```

### Tier 1: Pure Protocol & Chunker Tabular Tests
- **Scope**: Packet deserialization, serialization, checksum computation, bitwise mirroring, sliding-window chunking, and monitor payload decoding.
- **Characteristics**: 100% in-memory, deterministic, zero network or filesystem I/O, runs in sub-seconds.
- **Target packages**: [`internal/serial/chunker/lcn`](../internal/serial/chunker/lcn/lcn_test.go), [`pkg/serial/chunker`](../pkg/serial/chunker/chunker_test.go), [`internal/monitor`](../internal/monitor).

### Tier 2: In-Memory Component Fakes
- **Scope**: Testing [`serial.Port`](../pkg/serial/port.go) and [`mqtt.Broker`](../pkg/broker/mqtt/mqttBroker.go) coordination without real USB hardware or external brokers.
- **Fixtures**: Use `net.Pipe()` or in-memory channel-based serial mocks, and [`mqtttest.NewBroker()`](../pkg/broker/mqtt/mqtttest/simulator.go) with [`mqtttest.Simulator`](../pkg/broker/mqtt/mqtttest/simulator.go) to verify MQTT subscriptions, reconnect re-subscriptions, publishing, and context cancellation.

### Tier 3: Container Integration Tests
- **Scope**: Live end-to-end MQTT connectivity using an external Mosquitto broker.
- **Execution**: Run on demand or in CI workflows (`make test`). Bypasses sandbox only if running containerized daemon tests.

---

## Mandatory Test Mechanics

1. **Black-Box Test Packages**: Always define tests in a separate test package (`package <pkg>_test`, e.g., `package lcn_test`, `package chunker_test`). Never access unexported internals directly; test the public contract.
2. **Parallel Execution (`t.Parallel()`)**:
   - Every top-level test function must invoke `t.Parallel()`.
   - Every subtest within table loops must invoke `t.Parallel()`.
   ```go
   func TestDeserialize(t *testing.T) {
       t.Parallel()
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               t.Parallel()
               pkt, err := lcn.Deserialize(tt.input)
               // ...
           })
       }
   }
   ```
3. **Standard Test Context (`t.Context()`)**:
   - When tests require a `context.Context`, always call `t.Context()`.
   - Do not use `context.Background()` or `context.TODO()` inside tests (enforced by `usetesting.context-background: true` in `.golangci.yml`).
4. **Assertion Library Standards**:
   - Use `github.com/stretchr/testify/require` for fatal preconditions (e.g., verifying `require.NoError(t, err)` or `require.NotNil(t, pkt)` before inspecting fields).
   - Use `github.com/stretchr/testify/assert` for non-fatal value assertions.
   - Respect parameter order: `assert.Equal(t, expected, actual)`.
