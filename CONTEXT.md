# reverseLCN Ubiquitous Language & Domain Glossary

This document defines the ubiquitous language, protocol packet anatomy, and component terminology across the `reverseLCN` codebase.

---

## Hardware & Physical Bus Domain

### `LCN-PKU`
The **Local Control Network Parameterization and Coupling Unit (PKU)** is an official hardware interface module manufactured by Issendorff KG. It couples an installation's LCN bus to a PC over a USB connection. Internally, it presents itself to the operating system as a standard USB serial CDC/ACM or FTDI device (typically `/dev/ttyUSB0` or `/dev/ttyACM0`) operating at **9600 baud, 8 data bits, 1 stop bit, no parity**. The serial protocol is proprietary and binary.

### Physical LCN Frame Format
Every LCN message on the serial bus is a binary packet of 6, 8, 12, or 20 bytes. There are no other valid packet lengths on the physical wire.

| Byte Index | Field | Description | Framing Rules |
|:---:|:---:|---|---|
| `0` | **`SOURCE`** (`Src`) | Originating module ID | Bit-reversed on the wire (see `mirrorSrc`). |
| `1` | **`INFO`** (`Info`) | Frame metadata & length code | Bits 2-3 (`0x0C`) encode payload length. |
| `2` | **`CHECKSUM`** (`Checksum`) | Rotated-sum checksum | Computed across all bytes except byte 2 (see `calcChecksum`). |
| `3` | **`SEGMENT`** (`Seg`) | Destination segment ID | `0` denotes the local bus segment; non-zero targets remote segments. |
| `4` | **`DESTINATION`** (`Dst`) | Target module or group ID | Target ID `4` is conventionally the PC/Display coupler; IDs `5-254` are modules. |
| `5` | **`COMMAND`** (`Cmd`) | LCN operational command code | Determines payload interpretation (e.g., `0x13` relay, `0x68` status report). |
| `6-19` | **`PAYLOAD`** (`Payload`) | Optional command arguments | Length varies: 0 bytes (len 6), 2 bytes (len 8), 6 bytes (len 12), or 14 bytes (len 20). |

### Bit-Mirrored Source (`mirrorSrc`)
On the wire, the source address byte (`buf[0]`) has its bits reversed by the hardware/coupler: bit 0 swaps with bit 7, bit 1 with bit 6, bit 2 with bit 5, and bit 3 with bit 4.
For example, a source module ID of `1` (`0b00000001`) is represented on the wire as `0x80` (`0b10000000`).
The helper function [`mirrorSrc`](file:///home/kasch/projects/go/reverseLCN/internal/serial/chunker/lcn/lcn.go#L148) performs this 8-bit mirror transformation bidirectionally:
```go
func mirrorSrc(in byte) byte {
    src := byte(0)
    for p := 0; p < 8; p++ {
        src <<= 1
        src += (in & (1 << p) >> p)
    }
    return src
}
```

### `Info` Length Mask (`0x0C`)
The `Info` byte (`buf[1]`) carries operational flags, but bits 2 and 3 (`info & 0x0C`) specifically encode the total expected packet length:
- `0b00` (`lenCode6`): 6 bytes total (0 payload bytes)
- `0b01` (`lenCode8`): 8 bytes total (2 payload bytes)
- `0b10` (`lenCode12`): 12 bytes total (6 payload bytes)
- `0b11` (`lenCode20`): 20 bytes total (14 payload bytes)

Any other length code or buffer length mismatch indicates a corrupted or invalid frame.

### Rotated-Sum Checksum (`calcChecksum`)
Although vendor documentation references a CRC, reverse-engineering reveals the LCN checksum is a rotated modular sum.
Byte index `2` is skipped during computation. For every other byte `b` in the frame:
1. `tmp = int(b) + int(checksum)`
2. `tmp2 = ((tmp & 0x7F) << 2) | ((tmp & 0x180) >> 7)` (7-bit low shift left by 2, bits 7-8 shifted right by 7)
3. If `tmp2 > 0xFF`, `tmp2 -= 0xFF` (modular reduction by 255)
4. `checksum = byte(tmp2)`

The calculated checksum must exactly match byte `2` of incoming packets during deserialization, and is set into byte `2` during serialization.

---

## Protocol & Chunking Domain

### [`packet.Packet`](file:///home/kasch/projects/go/reverseLCN/pkg/serial/chunker/packet/packet.go#L10)
An interface representing any discrete protocol packet capable of serialization and human-readable formatting:
- `Serialize() ([]byte, error)`: Converts the packet structure into raw wire bytes (calculating checksums and lengths).
- `ToString() string`: Detailed diagnostic string displaying all packet fields in hexadecimal.
- `ToNiceString() string`: Formatted concise summary emphasizing source, destination, command, and payload.

### [`lcn.Packet`](file:///home/kasch/projects/go/reverseLCN/internal/serial/chunker/lcn/lcn.go#L34)
The concrete Go struct representing an LCN frame:
```go
type Packet struct {
    Src      byte
    Info     byte
    Checksum byte
    Seg      byte
    Dst      byte
    Cmd      byte
    Payload  []byte
}
```

### [`packet.Deserializer`](file:///home/kasch/projects/go/reverseLCN/pkg/serial/chunker/packet/packet.go#L16)
A function signature `func(buf []byte) (Packet, error)` that attempts to parse a raw byte slice into a typed packet. [`lcn.Deserialize`](file:///home/kasch/projects/go/reverseLCN/internal/serial/chunker/lcn/lcn.go#L74) is the primary deserializer for LCN traffic.

### `ErrPacketIncomplete` vs `ErrPacketInvalid`
The chunking engine distinguishes between partial frames and corrupt data using wrapped sentinel errors:
- [`packet.ErrPacketIncomplete`](file:///home/kasch/projects/go/reverseLCN/pkg/serial/chunker/packet/packet.go#L6) (wrapped by `ErrLcnPacketIncomplete`): The buffer contains fewer bytes than the minimum packet length (6) or fewer bytes than indicated by the `Info` length code. The chunker must **continue accumulating** incoming bytes.
- [`packet.ErrPacketInvalid`](file:///home/kasch/projects/go/reverseLCN/pkg/serial/chunker/packet/packet.go#L7) (wrapped by `ErrLcnPacketInvalid` and `ErrLcnPacketInvalidChecksum`): The buffer contains an illegal length, an invalid length code, or a mismatched checksum. The chunker must **advance its search window by 1 byte** to discard noise and re-synchronize to the next valid frame boundary.

### [`chunker.Chunker`](file:///home/kasch/projects/go/reverseLCN/pkg/serial/chunker/chunker.go#L19) & [`chunker.EjectFunc`](file:///home/kasch/projects/go/reverseLCN/pkg/serial/chunker/chunker.go#L17)
- `EjectFunc`: Callback `func(packet.Packet)` invoked when a valid packet is completely assembled.
- `Chunker`: Interface `Collect(buf []byte, eject EjectFunc)`. Manages an internal `bytes.Buffer`, slicing through serial chunks, handling fragmentation across OS read boundaries, and discarding corrupt noise bytes until a clean frame emerges.

---

## Serial & Broker Domain

### [`serial.Port`](file:///home/kasch/projects/go/reverseLCN/pkg/serial/port.go#L18)
The serial communication abstraction managing asynchronous USB tty read and write loops:
- `Run(ctx context.Context, cancel context.CancelFunc, eject chunker.EjectFunc)`: Spawns the background reader goroutine (feeding raw bytes into the chunker) and the writer goroutine.
- `Send(buf []byte)`: Non-blocking enqueue to `sendQueue chan []byte` for transmission onto the physical wire.

### [`broker.Broker`](file:///home/kasch/projects/go/reverseLCN/pkg/broker/broker.go#L5) & [`broker.Topic`](file:///home/kasch/projects/go/reverseLCN/pkg/broker/broker.go#L10)
High-level pub/sub abstraction decoupling the application from specific messaging implementations:
- `Broker`: Controls lifecycle (`Run`) and topic resolution (`Topic(name)`).
- `Topic`: Provides typed publish/subscribe primitives:
  - `Publish(interface{})`: Encodes data to JSON and publishes with QoS 1.
  - `PublishString(string)`: Publishes raw text.
  - `Subscribe(hint interface{}, callback CallbackFunction)`: Subscribes with QoS 0, unmarshals incoming JSON payloads into the type supplied by `hint` (using reflection), and triggers the callback.
- [`nullBroker`](file:///home/kasch/projects/go/reverseLCN/pkg/broker/null/nullBroker.go#L10): A no-op implementation activated when MQTT is disabled in configuration.
- [`mqttBroker`](file:///home/kasch/projects/go/reverseLCN/pkg/broker/mqtt/mqttBroker.go#L24): Paho MQTT implementation connecting to external brokers (e.g., Mosquitto).

---

## Monitoring & Telemetry Domain

### [`monitor.DataStore`](file:///home/kasch/projects/go/reverseLCN/internal/monitor/datastore.go#L14)
A thread-safe in-memory cache protected by a `sync.Mutex`. Stores unique messages keyed by their string representation, tracking total occurrences (`times`) and the `lastSeen` timestamp.

### Payload Decoders
Specialized parsers in [`internal/monitor/parsers.go`](file:///home/kasch/projects/go/reverseLCN/internal/monitor/parsers.go) that convert raw byte payloads into descriptive human telemetry:
- [`decodeRelais`](file:///home/kasch/projects/go/reverseLCN/internal/monitor/parsers.go#L34) (Command `0x13`): Interprets 8 relay bitmask channels on actuator modules. Determines whether a channel is commanded to `FORCE ON`, `FORCE OFF`, or `TOGGLE`.
- [`decodeStatusReport`](file:///home/kasch/projects/go/reverseLCN/internal/monitor/parsers.go#L59) (Command `0x68`): Decodes module status broadcasts sent to the host display (ID `4`), identifying active relay channels.
- [`decodeStatusQuery`](file:///home/kasch/projects/go/reverseLCN/internal/monitor/parsers.go#L75) (Command `0x6E`): Parses status inquiries (`0xFB` for query, `0x7B` for report) and maps active bit channels to friendly appliance labels.
- [`idMap`](file:///home/kasch/projects/go/reverseLCN/internal/monitor/mappings.go#L6) & [`moduleOutputs`](file:///home/kasch/projects/go/reverseLCN/internal/monitor/mappings.go#L18): Static lookup tables that map numeric module IDs (e.g., `33` -> `R8H 33 - Licht A`) and channel indices (e.g., `33-0` -> `Strahler Wohnen/Essen`) to physical installation names.
- [`mapIfPossible`](file:///home/kasch/projects/go/reverseLCN/internal/monitor/mappings.go#L51): Formats an ID or fallback integer into a readable name.
