# reverseLCN System Architecture

This document describes the architectural layout, package dependency graph, bidirectional communication flows, sliding-window chunker resynchronization mechanics, and container build strategy of `reverseLCN`.

---

## Dual-Binary Package DAG

The repository compiles two independent CLI binaries from [`cmd/`](../cmd):
1. `lcn2mqtt`: Production USB serial <-> MQTT gateway bridge.
2. `lcnMonitor`: Diagnostic bus observer and packet decoding subscriber.

The architecture strictly enforces package boundaries: generic utility packages located under [`pkg/`](../pkg) never import application packages under [`cmd/`](../cmd), [`config/`](../config), or [`internal/`](../internal).

```mermaid
flowchart TD
    subgraph CLI["CLI Entrypoints (cmd/)"]
        lcn2mqtt["cmd/lcn2mqtt"]
        lcnMonitor["cmd/lcnMonitor"]
    end

    subgraph Internal["Application Internals (internal/ & config/)"]
        cfg["config (Viper & Config structs)"]
        cmdInit["internal/cmd (Init, Signals, Logger)"]
        monitor["internal/monitor (DataStore, Parsers, Mappings)"]
        lcnProto["internal/serial/chunker/lcn (Packet, Deserialize, Checksum)"]
    end

    subgraph Pkg["Reusable Platform Libraries (pkg/)"]
        logPkg["pkg/log (Zap SugaredLogger)"]
        hiddenPkg["pkg/hidden (Secret String & URL Masking)"]
        mqttPkg["pkg/broker/mqtt (Paho MQTT Client & TLS)"]
        serialPkg["pkg/serial (Port, Options, Reader/Writer)"]
        chunkerPkg["pkg/serial/chunker (Chunker interface & sliding-buffer)"]
        packetPkg["pkg/serial/chunker/packet (Packet & Deserializer interfaces)"]
        plainPkg["pkg/serial/chunker/plain (Hex/Raw fallback Packet)"]
    end

    lcn2mqtt --> cmdInit
    lcn2mqtt --> mqttPkg
    lcn2mqtt --> serialPkg
    lcn2mqtt --> lcnProto
    lcn2mqtt --> packetPkg
    lcn2mqtt --> logPkg

    lcnMonitor --> cmdInit
    lcnMonitor --> mqttPkg
    lcnMonitor --> monitor
    lcnMonitor --> lcnProto
    lcnMonitor --> logPkg

    cmdInit --> cfg
    cmdInit --> mqttPkg
    cmdInit --> logPkg

    cfg --> hiddenPkg
    cfg --> logPkg

    monitor --> lcnProto
    monitor --> logPkg

    lcnProto --> packetPkg
    lcnProto --> logPkg

    serialPkg --> chunkerPkg
    serialPkg --> logPkg

    chunkerPkg --> packetPkg
    chunkerPkg --> logPkg

    plainPkg --> packetPkg

    mqttPkg --> logPkg

    classDef pkgClass fill:#e1f5fe,stroke:#0288d1,stroke-width:2px;
    classDef internalClass fill:#fff3e0,stroke:#f57c00,stroke-width:2px;
    classDef cliClass fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px;

    class lcn2mqtt,lcnMonitor cliClass;
    class cfg,cmdInit,monitor,lcnProto internalClass;
    class logPkg,hiddenPkg,mqttPkg,serialPkg,chunkerPkg,packetPkg,plainPkg pkgClass;
```

---

## Bidirectional Bridge Sequence

[`cmd/lcn2mqtt`](../cmd/lcn2mqtt/lcn2mqtt.go) mediates between the physical serial connection (`LCN-PKU`) and an MQTT broker. Communication is fully bidirectional:

### 1. Bus -> MQTT (Inbound Telemetry)
The physical bus transmits electrical pulses read by the PKU USB adapter. Incoming bytes are chunked, validated, deserialized into an [`lcn.Packet`](../internal/serial/chunker/lcn/lcn.go), and published as JSON to `lcn/segment/<seg>/target/<dst>/`.

### 2. MQTT -> Bus (Outbound Commands)
An external client or smart-home controller publishes an LCN packet JSON payload to `lcn/in`. The bridge receives the message, unmarshals the JSON into an `lcn.Packet`, computes the rotated-sum checksum and mirrored source byte, and queues the binary buffer to the serial writer.

```mermaid
sequenceDiagram
    autonumber
    participant Bus as Physical LCN Bus
    participant PKU as LCN-PKU (USB Serial)
    participant Port as pkg/serial (Port & Chunker)
    participant Bridge as cmd/lcn2mqtt
    participant Broker as MQTT Broker
    participant Client as Home Assistant / External Client

    Note over Bus,Broker: Inbound Flow: Bus -> MQTT
    Bus->>PKU: Electrical Bus Frame (6-20 bytes)
    PKU->>Port: Raw serial bytes (/dev/ttyUSB0)
    Port->>Port: Chunker.Collect() re-assembles frame
    Port->>Bridge: EjectFunc(pkt *lcn.Packet)
    Bridge->>Bridge: Format topic "lcn/segment/<seg>/target/<dst>/"
    Bridge->>Broker: Publish JSON {"Src": ..., "Dst": ..., "Cmd": ..., "Payload": "..."}
    Broker->>Client: Deliver telemetry message

    Note over Bus,Broker: Outbound Flow: MQTT -> Bus
    Client->>Broker: Publish to "lcn/in" (JSON packet)
    Broker->>Bridge: MQTT message callback (lcn.Packet)
    Bridge->>Bridge: pkt.Serialize() (mirrorSrc + calcChecksum)
    Bridge->>Port: port.Send(ctx, wireBytes) (non-blocking FIFO queue)
    Port->>PKU: port.Write(wireBytes)
    PKU->>Bus: Electrical Bus Transmission
```

---

## Sliding-Window Chunker Resynchronization

Because serial connections over USB can experience noise, frame fragmentation, or byte loss across OS `read()` boundaries, [`pkg/serial/chunker/chunker.go`](../pkg/serial/chunker/chunker.go) implements a sliding-window resynchronization state machine.

### State Transitions

1. **Ingest**: Raw bytes read from the serial port are appended directly to the internal `bytes.Buffer`.
2. **Length Guard**: If `buffer.Len() < minLength` (where `minLength = 6` for LCN), the chunker yields back to the serial reader to await more data.
3. **Speculative Deserialization**: The chunker presents current buffer bytes to the registered [`packet.Deserializer`](../pkg/serial/chunker/packet/packet.go).
4. **Resolution**:
   - **`ErrPacketIncomplete`**: The packet header specifies a larger length than currently buffered (e.g. `Info` specifies length 12 or 20, but only 8 bytes have arrived). The loop breaks without consuming bytes and waits for the next OS read (`continue read_loop`). If the serial read idle timeout (`100ms`) expires with no further bytes arriving, `Flush()` advances past leading stalled noise bytes to recover any valid shorter frame hidden behind them and resets trailing sub-minimum fragments.
   - **`ErrPacketInvalid` or Checksum Error**: The buffer contains corrupt framing, an illegal length code, or an invalid checksum. The chunker calls `buffer.Next(1)`, **discarding exactly 1 byte** from the head of the buffer to re-test the subsequent byte as a potential frame start.
   - **Success**: A clean, checksum-verified prefix (`buf[:expectedLen]`) is assembled and returns `(pkt, consumed, nil)`. The chunker invokes [`eject(pkt)`](../pkg/serial/chunker/chunker.go) and advances by **`buffer.Next(consumed)`**, preserving any trailing bytes in `c.buffer` for subsequent loop iterations.

```mermaid
flowchart TD
    Start["Receive byte buffer from serial.Read()"] --> CheckRead{"n > 0 bytes read?"}
    CheckRead -- Yes --> Append["Append bytes to c.buffer"]
    CheckRead -- "No (100ms idle timeout)" --> Flush["c.buffer: Flush stalled candidates & sub-min tail"]
    Append --> CheckLen{"c.buffer.Len() >= minLength (6)?"}
    
    CheckLen -- No --> Wait["Exit inner loop; await next serial read"]
    CheckLen -- Yes --> Deser["Call deserializer(c.buffer.Bytes())"]
    
    Deser --> Result{"Evaluate deserializer result"}
    
    Result -- "ErrPacketIncomplete" --> Wait
    Result -- "ErrPacketInvalid / Bad Checksum" --> Log["Log error & buffer hex"]
    Log --> Shift["c.buffer.Next(1) (Shift window 1 byte)"]
    Shift --> CheckLen
    
    Result -- "Success (*lcn.Packet, consumed)" --> Eject["eject(pkt) (Invoke callback)"]
    Eject --> Advance["c.buffer.Next(consumed)"]
    Advance --> CheckLen
```

---

## Container Build Architecture (`FROM scratch`)

The container images for `lcn2mqtt` are defined in [`Dockerfile`](../Dockerfile) and [`build/lcn2mqtt/Dockerfile`](../build/lcn2mqtt/Dockerfile). The build uses a strict multi-stage workflow:

### Stage 1: Static Builder (`golang:1.27-alpine`)
- Runs natively on `$BUILDPLATFORM` (`FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder`) and cross-compiles for `$TARGETOS`/`$TARGETARCH`/`$TARGETVARIANT` (`linux/amd64`, `linux/arm64`, `linux/arm/v7`, `linux/arm/v6`).
- Installs minimal toolchain requirements (`ca-certificates`, `tzdata`) and creates an unprivileged user (`appuser`, `UID=10001`).
- Compiles a fully static binary (`CGO_ENABLED=0`) with Go build cache mounting and stamps `main.Version` via `-ldflags`:
  ```bash
  CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} \
  go build -v -ldflags "-w -s -X main.Version=${APP_VERSION}" -o /dist/lcn2mqtt ./cmd/lcn2mqtt
  ```

### Stage 2: Runtime (`FROM scratch`)
- Copies time zone data from `/usr/share/zoneinfo`.
- Copies trusted SSL/TLS root certificates from `/etc/ssl/certs/ca-certificates.crt` (required for TLS MQTT broker connections).
- Copies `/etc/passwd` and `/etc/group` from builder and switches to `USER appuser:appuser` (`10001:10001`).
- Copies the static binary `/dist/lcn2mqtt` into root `/`.
- Copies default configuration from `/build/config/config.yml` into `/config/`.
- Configures `ENTRYPOINT ["/lcn2mqtt"]`.

This produces a zero-footprint, hardened image containing no shell, package manager, or unnecessary OS binaries, minimizing attack surface and eliminating container base-image CVEs.

### Helm Chart Packaging (`helm/charts/reverselcn`)
- [`helm/charts/reverselcn/`](../helm/charts/reverselcn/) packages `lcn2mqtt` as a non-root, privilege-dropped Kubernetes `DaemonSet` (`runAsNonRoot: true`, `runAsUser: 10001`, `supplementalGroups: [20]` for `dialout`, `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, `capabilities.drop: ["ALL"]`) using `updateStrategy.rollingUpdate` (`maxSurge: 0, maxUnavailable: 1`) and mounting the serial character device (`type: CharDevice`).
- Automatically injects `LCN_SERIAL_PORT` matching `serialDevice.mountPath` when `serialDevice.enabled` is `true` (unless explicitly overridden in `env`), supports `envFrom` and `extraVolumeMounts`/`extraVolumes` for credentials/TLS mounts, and recommends targeting the USB host node via `nodeSelector`.
- Published automatically to GitHub Pages via [`.github/workflows/release-chart.yml`](../.github/workflows/release-chart.yml).
