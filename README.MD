# reverseLCN

This application is built to reverse-engineer the LCN serial protocol and thus make the LCN bus smart home compatible just using a LCN-PKU, as this should always be available in any LCN installation. 

The LCN-PKU has an USB port which indeed is a serial port adapter to the LCN-bus. It can read and write data from/to the bus. The protocol is proprietary.

`lcn2mqtt` bridges the LCN bus via USB to a MQTT broker, and publishes all bus messages to a topic. It also can read packets to be published from another topic and publish those to the LCN bus, which enables full smart home connectivity.

# The LCN package format

| Bytes |    0   |   1  |     2    |    3    |      4      |    5    |   6-19  |
|:-----:|:------:|:----:|:--------:|:-------:|:-----------:|:-------:|:-------:|
| Usage | SOURCE | INFO | CHECKSUM | SEGMENT | DESTINATION | COMMAND | PAYLOAD |

* SOURCE - ID of the source, but somebody thought its a nice idea to mirror the bits... so 0x80 means 1, the bit pairs 0:7, 1:6, 2:5 and 3:4 switch positions
* INFO - only partially known. 
  * Bits masked by 0x0C seem to hold payload length information 
* CHECKSUM - although Issendorff claims this is a CRC, I was unable to find a fitting CRC mechanism. No CRC Polynom could be found.
* SEGMENT - Segment ID to send the message to
* DESTINATION - ID of the target device
* COMMAND - Target command to be executed
* PAYLOAD - Parameters for the command

# mqtt control example
```
pub lcn/in {\"Src\":1,\"Seg\":0,\"Dst\":33,\"Cmd\":19,\"Payload\":\"AIA=\"}
```

# Disclaimer
This is highly experimental. I test this with my own LCN bus system, but cannot guarantee that any other system works. There is a lot of 'magic' involved as I have no access to any official documentation from the vendor. Most is reverse engineered.

# Thanks and Sources
* Daniel Dallmann - [Yali](https://sourceforge.net/projects/yali/) for his checksum and some information on packet lengths
* Issendorff - [LCN Systembeschreibung](https://www.lcn.eu/?wpdmdl=8114)

# License 
GPLv3

---

## Kubernetes Deployment (Helm)

The chart under [`helm/charts/reverselcn/`](./helm/charts/reverselcn) deploys `lcn2mqtt` as an unprivileged `DaemonSet` mounting the USB serial adapter (`serialDevice.mountPath`, defaulting to `/dev/lcn-pku`):

- When `serialDevice.enabled` is `true`, the template automatically injects `LCN_SERIAL_PORT` matching `serialDevice.mountPath` unless explicitly overridden in `env`.
- On multi-node clusters, label the node physically attached to the LCN-PKU USB adapter and configure `nodeSelector` in Helm values so pods do not stall in `ContainerCreating` on nodes without the device:
  ```bash
  kubectl label nodes <usb-host> node-role.kubernetes.io/lcn-serial=true
  ```
  ```yaml
  nodeSelector:
    node-role.kubernetes.io/lcn-serial: "true"
  ```
- Pass MQTT credentials via `env` / `envFrom` (`LCN_MQTT_CLIENTID`, `LCN_MQTT_USERNAME`, `LCN_MQTT_PASSWORD`) and mount custom CA/client certificates using `extraVolumeMounts` / `extraVolumes` with `LCN_MQTT_TLS_*`.

---

## Documentation & Development

For deeper technical documentation, architecture blueprints, coding standards, and developer workflows, refer to:

- [Agent Guide (`AGENTS.md`)](./AGENTS.md): Project overview, linear-main workflow, environment prerequisites, and hazards.
- [Ubiquitous Language & Domain Glossary (`CONTEXT.md`)](./CONTEXT.md): LCN-PKU hardware, 6/8/12/20-byte packet anatomy, rotated checksum, and domain models.
- [System Architecture (`docs/ARCHITECTURE.md`)](./docs/ARCHITECTURE.md): Dual-binary architecture, package DAG, sequence diagrams, and sliding-window chunker state machine.
- [Coding Conventions (`docs/CONVENTIONS.md`)](./docs/CONVENTIONS.md): Go 1.27 idioms, binary bitwise masks, sentinel errors, and functional options.
- [Testing Standards (`docs/TESTING.md`)](./docs/TESTING.md): Output-oriented tabular tests, 3-tier testing model, and mandatory test mechanics.
- [Developer Toolchain (`docs/TOOLCHAIN.md`)](./docs/TOOLCHAIN.md): Makefile targets, isolated tool modules, vulnerability scanning, and licensing.
