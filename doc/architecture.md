# Architecture

This document describes how the firmware is built up, how a command travels from the host to a relay,
and how the image gets from a commit to a release. For installing and using the firmware see the
[README](../README.md).

## Overview

The firmware runs on a Raspberry Pi Pico (RP2040) plugged onto a Waveshare Pico Relay B. It shows up on the
host as a USB serial port. Programs on the host use the [norn](https://github.com/nornforge/norn) package to
send commands; the firmware switches the relays and answers each command.

```mermaid
flowchart LR
    subgraph host["Host"]
        cli["norn CLI"]
        server["norn server (REST)"]
        client["norn-client / HTTP clients"]
        client -- HTTP --> server
    end

    subgraph board["Waveshare Pico Relay B"]
        subgraph pico["Raspberry Pi Pico (RP2040)"]
            fw["norn firmware"]
            boot["USB bootloader (ROM)"]
        end
        relays["8 relays<br/>GP21 … GP14"]
        rgb["RGB status LED (WS2812)<br/>GP13"]
    end

    cli -- "USB CDC serial<br/>/dev/ttyACM*" --> fw
    server -- "USB CDC serial" --> fw
    fw -- GPIO --> relays
    fw -- "PIO, one wire" --> rgb
    fw -. "bootloader command" .-> boot
```

Only one program should use the serial port at a time. Several clients share the board through the REST server
of the norn CLI.

## Packages

The code that decides what happens is kept free of TinyGo's `machine` package, so its tests run on the
development machine without hardware (also with plain `go test`). Only `firmware/` touches the hardware.

```mermaid
flowchart TD
    main["firmware<br/>main.go, watchdog.go<br/>pico.go / xiao-rp2040.go"]
    controller["controller<br/>executes commands"]
    channel["channel<br/>one relay output"]
    status["status<br/>what the LED shows"]
    norn["github.com/nornforge/norn/pkg/norn<br/>frames, Command, Response"]
    machine["machine<br/>TinyGo, hardware"]
    ws2812["tinygo.org/x/drivers/ws2812"]

    main --> controller
    main --> channel
    main --> status
    main --> norn
    main --> machine
    main --> ws2812
    controller --> channel
    controller --> norn

    classDef hw fill:#fde2e2,stroke:#c0392b
    classDef ext fill:#e8eefc,stroke:#3b5bdb
    class machine,ws2812 hw
    class norn ext
```

| Package       | Responsibility                                                                                     |
| ------------- | -------------------------------------------------------------------------------------------------- |
| `firmware/`   | Wires everything up: pins, serial line, LED driver, watchdog, bootloader. The only `machine` user. |
| `controller/` | Executes a `norn.Command` on the channels and builds the `norn.Response`.                          |
| `channel/`    | A relay output that remembers the state it was switched to. Drives a `channel.Pin`.                |
| `status/`     | Which color the LED shows for which state, how long, and at which brightness.                      |

Interfaces keep the hardware out of these packages:

| Interface        | Needed by           | Satisfied on the board by | Satisfied in tests by |
| ---------------- | ------------------- | ------------------------- | --------------------- |
| `channel.Pin`    | `channel.Channel`   | `machine.Pin`             | a fake pin            |
| `status.Writer`  | `status.Light`      | `ws2812.Device`           | a fake LED            |
| `func()`         | `controller` (bootloader) | `enterBootloader` in `firmware/` | a counter     |
| `func(uint8)`    | `controller` (brightness) | `status.Light.SetBrightness`    | a recorder    |

### Board variants

The pin mapping is chosen with the TinyGo target. Both files define `led`, `rgbPin` and `Channels`.

| Target        | File                      | Channels | Note                                                  |
| ------------- | ------------------------- | -------- | ----------------------------------------------------- |
| `pico`        | `firmware/pico.go`        | 8        | Waveshare Pico Relay B; the released image            |
| `xiao-rp2040` | `firmware/xiao-rp2040.go` | 4        | not pin compatible with the Relay B; built in CI only |

## Runtime

Two goroutines run after start-up. One reads frames from the serial line, the other executes them. The
command loop also wakes up every 50 ms on its own, so the LED can return to green and the watchdog is fed
while no command arrives.

```mermaid
flowchart LR
    usb[("USB CDC<br/>receive buffer")]

    subgraph reader["goroutine readFromSerial"]
        parse["norn.Command.Parse"]
    end

    subgraph loop["main goroutine: command loop"]
        sel{"select"}
        handle["controller.Handle"]
        reply["write response"]
        led["light.Show(Busy / Failed)"]
        tick["light.Tick"]
        feed["feedWatchdog"]
    end

    usb --> parse
    parse -- "chan norn.Command" --> sel
    parse -- "parse error:<br/>error response" --> out[("USB CDC<br/>send")]
    ticker(["ticker, 50 ms"]) --> sel
    sel -- command --> handle --> reply --> led --> tick
    sel -- tick --> tick
    tick --> feed --> sel
    reply --> out
```

Running the commands in a single goroutine means the controller, the channels and the LED need no locking.

### A command from the host to the relay

```mermaid
sequenceDiagram
    autonumber
    participant Host as norn CLI
    participant Reader as readFromSerial
    participant Loop as command loop
    participant Ctrl as controller
    participant Ch as channel
    participant LED as status.Light

    Host->>Reader: frame {"type":2,"channel":3,"status":true}
    Reader->>Reader: check marker, length, CRC-32C
    Reader->>Loop: norn.Command via channel
    Loop->>Ctrl: Handle(command)
    Ctrl->>Ch: Set(true)
    Ch->>Ch: pin.Set(true), remember state
    Ctrl-->>Loop: Response{isOk, channel 3, status true}
    Loop-->>Host: response frame
    Loop->>LED: Show(Busy), blue for 150 ms
    Note over Loop,LED: later ticks return the LED to Idle (green)
```

If the frame cannot be parsed, `readFromSerial` answers with an error frame itself and the command never
reaches the loop. If the controller rejects the command (unknown type, channel out of range, brightness above
255), the response has `isOk: false` and the LED turns red for one second.

The frame layout and the command types are described in the README under
[Protocol](../README.md#protocol).

### Status LED

```mermaid
stateDiagram-v2
    direction LR
    [*] --> Idle: start-up
    Idle --> Busy: command succeeded
    Idle --> Failed: command failed
    Busy --> Idle: after 150 ms
    Failed --> Idle: after 1 s
    Busy --> Busy: command succeeded
    Busy --> Failed: command failed
    Failed --> Busy: command succeeded
    Failed --> Failed: command failed
    Idle --> Updating: bootloader command
    Busy --> Updating: bootloader command
    Failed --> Updating: bootloader command
    Updating --> [*]: bootloader running

    Idle: Idle (green)
    Busy: Busy (blue)
    Failed: Failed (red)
    Updating: Updating (magenta)
```

The hold times are checked by `Light.Tick`, which the command loop calls after every command and every 50 ms.
The brightness (default 16 of 255) scales all colors and does not change the state.

## Relay safety

Off is the state the firmware falls back to. Several places make sure a relay is never switched on by
accident:

```mermaid
flowchart TD
    power(["power-up, reset or watchdog restart"]) --> low["pin.Low()<br/>before the pin becomes an output"]
    low --> out["pin configured as output"]
    out --> new["channel.New switches the channel off"]
    new --> run["command loop running,<br/>watchdog started (2 s)"]
    run -- "set command" --> on["relay follows the command"]
    on --> run
    run -- "loop stops, e.g. panic" --> wd["watchdog expires"] --> power
    run -- "bootloader command" --> alloff["controller.AllOff"]
    alloff --> magenta["LED magenta"] --> stop["watchdog stopped"] --> rom(["USB bootloader (ROM)"])
    rom -- "new image copied" --> power
```

- **Start-up:** the level of each relay pin is set low before the pin is switched to an output, so the relays
  do not see a pulse. `channel.New` then records the state as off.
- **Watchdog:** it restarts the firmware if the command loop has not run for two seconds. After the restart all
  relays are off.
- **Bootloader:** all relays are switched off before the bootloader is entered, because nothing controls them
  while it runs. The watchdog is stopped first, since the bootloader does not feed it.
- **Host gone:** if the host program exits or USB is unplugged while the board stays powered, the relays keep
  their state. The firmware does not watch for the host: a CLI call ends right after its command, and the
  relays are expected to stay as it left them.

The resulting behaviour is summarised in the README under [Relay states](../README.md#relay-states).

## Build and release

```mermaid
flowchart LR
    subgraph dev["Dev container"]
        code["code change"] --> test["tinygo test ./..."]
        test --> build["tinygo build -target=pico"]
        build --> uf2["firmware.uf2<br/>copied onto the Pico by hand"]
    end

    subgraph ci["GitHub Actions"]
        pr(["push to main / pull request"]) --> ciJob["go.yml:<br/>gofmt, tests,<br/>build-release.sh v0.0.0-ci,<br/>XIAO build"]
        ciJob --> artifact["workflow artifact<br/>firmware_pico"]

        tag(["tag vX.Y.Z"]) --> relJob["release.yml:<br/>tests,<br/>build-release.sh vX.Y.Z"]
        relJob --> changelog["chglog:<br/>changelog from<br/>Conventional Commits"]
        relJob --> attest["build provenance<br/>attestation"]
        changelog --> release["GitHub release"]
        attest --> release
    end

    code -. "pull request" .-> pr
```

`scripts/build-release.sh <version>` produces the same files locally and in both workflows:

| File                                   | Content                                                             |
| -------------------------------------- | ------------------------------------------------------------------- |
| `norn-firmware_pico_<version>.uf2`     | the firmware image, version injected with `-ldflags -X main.Version` |
| `norn-firmware_pico_<version>.cdx.json`| CycloneDX 1.5 SBOM: linked Go modules, TinyGo, Go standard library   |
| `checksums.txt`                        | SHA-256 of image and SBOM                                           |

Only tags of the form `vX.Y.Z` start a release. The versions of Go, TinyGo and the norn CLI are pinned in the
dev container and in the workflows and are kept in sync by hand; the actions are pinned to commit hashes and
updated by Dependabot.
