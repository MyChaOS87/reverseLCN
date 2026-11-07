# reverseLCN Coding Idioms & Conventions

This document records the coding style, Go language standards, protocol bitwise rules, error handling, and commit conventions across `reverseLCN`.

---

## Go 1.27 Modernization & Language Idioms

`reverseLCN` targets Go 1.27 and relies on modern standard library capabilities:

- **AST Modernization via `go fix`**: Run `go fix ./...` to adopt standard library improvements and AST migrations. `make lint` enforces `go fix -diff ./...`.
- **Standard Library Primitives**: Use standard library primitives over external packages:
  - Use stdlib `errors` (`errors.Is`, `errors.As`, `fmt.Errorf`) instead of `github.com/pkg/errors` (blocked by `gomodguard_v2`).
  - Use stdlib `slices` (e.g., [`slices.SortFunc`](../internal/monitor/datastore.go#L58)) and built-in `min`/`max` functions instead of custom loops.
  - Test context creation must use `t.Context()` instead of `context.Background()` or `context.TODO()` in tests (enforced by `usetesting`).
- **Loop Variable Scoping**: Go 1.22+ per-iteration variable scoping is active. Do not add redundant `tt := tt` redeclarations in new subtests.

---

## Binary Protocol Constants & Bitwise Operations

All binary protocol manipulation for LCN framing must adhere to clear bitwise mask conventions:

### Length and Framing Constants
Constants must use explicit binary representation (`0b...`) or hex where they represent bit patterns, and decimal where they represent byte counts:

```go
const (
    minLcnPacketLength = 6
    lcnPacketLength8   = 8
    lcnPacketLength12  = 12
    lcnPacketLength20  = 20

    infoLengthMask  = 0x0C // 0b00001100
    infoLengthShift = 2

    lenCode6  = 0b00
    lenCode8  = 0b01
    lenCode12 = 0b10
    lenCode20 = 0b11
)
```

### Bitwise Masking and Clearing
When updating packet headers, clear target bits using bit-clear `&^` before OR-ing shifted values:
```go
// Clear the 2 length bits (mask 0x0C) and inject the 2-bit length code shifted left by 2:
buf[1] = buf[1]&^infoLengthMask | (code << infoLengthShift)
```

When decoding bit flags:
```go
// Extract length code from bits 2-3:
lengthCode := (info & infoLengthMask) >> infoLengthShift
```

When checking channel bits (0-7):
```go
func testDigit(b byte, out int) bool {
    if out < 0 || out >= 8 {
        return false
    }
    return b&(1<<out) != 0
}
```

---

## Sentinel Error Wrapping (`%w`)

To allow the chunker and protocol layers to reliably inspect errors with `errors.Is`, domain errors must wrap base sentinel errors using `%w`:

### Base Sentinels ([`pkg/serial/chunker/packet`](../pkg/serial/chunker/packet/packet.go#L5))
```go
var (
    ErrPacketIncomplete = errors.New("Packet Incomplete")
    ErrPacketInvalid    = errors.New("Packet Invalid")
)
```

### Protocol-Specific Wrapped Errors ([`internal/serial/chunker/lcn`](../internal/serial/chunker/lcn/lcn.go#L26))
```go
var (
    ErrLcnPacketIncomplete      = fmt.Errorf("%w: LCN Packet to short", packet.ErrPacketIncomplete)
    ErrLcnPacketInvalid         = fmt.Errorf("%w: LCN Packet Invalid", packet.ErrPacketInvalid)
    ErrLcnPacketInvalidChecksum = fmt.Errorf("%w: LCN Checksum invalid", packet.ErrPacketInvalid)
)
```

### Error Inspection
Always inspect errors using `errors.Is`:
```go
if errors.Is(err, packet.ErrPacketIncomplete) {
    // Await more data from serial buffer
    return
}
```
Never compare errors via direct string comparison or direct equality (`err == ...`).

---

## Functional Options Pattern

Packages configuring I/O devices or network clients must use functional options:

### Serial Port Configuration ([`pkg/serial`](../pkg/serial/options.go#L10))
```go
type Option func(*Config)

func PortName(portName string) Option {
    return func(c *Config) { c.portName = portName }
}

func BaudRate(baudRate int) Option {
    return func(c *Config) { c.baudRate = baudRate }
}

func Deserializer(deserializer packet.Deserializer) Option {
    return func(c *Config) { c.deserializer = deserializer }
}
```

Usage in [`cmd/lcn2mqtt`](../cmd/lcn2mqtt/lcn2mqtt.go#L30):
```go
port := serial.NewPort(
    serial.BaudRate(cfg.Serial.BaudRate),
    serial.PortName(cfg.Serial.Port),
    serial.Deserializer(lcn.Deserialize),
)
```

### MQTT Broker Configuration ([`pkg/broker/mqtt`](../pkg/broker/mqtt/options.go#L9))
```go
func Broker(broker string) Option {
    return func(c *Config) {
        c.clientOptions.AddBroker(broker)
    }
}

func TLS(tls *tls.Config) Option {
    return func(c *Config) {
        c.clientOptions.TLSConfig = tls
    }
}
```

Constructors must instantiate sensible defaults before applying variadic options.

---

## Configuration via Viper (`LCN_*`)

Configuration management in [`config/config.go`](../config/config.go#L38) uses Viper with explicit hierarchical mapping:

```go
const envPrefix = "LCN"

func LoadConfig(filename string) (*viper.Viper, error) {
    v := viper.New()
    v.SetConfigName(filename)
    v.AddConfigPath(".")
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    v.SetEnvPrefix(envPrefix)
    v.AutomaticEnv()
    // ...
}
```

### Environment Variable Rules
Nested configuration keys map to environment variables prefixed with `LCN_`:
- `serial.port` -> `LCN_SERIAL_PORT`
- `serial.baudRate` -> `LCN_SERIAL_BAUDRATE`
- `mqtt.broker` -> `LCN_MQTT_BROKER`
- `mqtt.rootTopic` -> `LCN_MQTT_ROOTTOPIC`
- `mqtt.enabled` -> `LCN_MQTT_ENABLED`
- `logger.level` -> `LCN_LOGGER_LEVEL`
- `logger.development` -> `LCN_LOGGER_DEVELOPMENT`

---

## Conventional Commit Guidelines

All commits must follow the workspace-standard conventional commit format:

```
feat[(<topic>)]: <description>
chore[(<topic>)]: <description>
bugfix[(<topic>)]: <description>
```

- Write in the active present tense ("add serial test" not "added serial test").
- Use a single subject line; if bullets are needed in the body, use `*`.
- When typing commands containing literal `$` in commit messages or git arguments, escape backticks or shell variables properly.
