# Norn Firmware

[![TinyGo](https://github.com/nornforge/firmware/actions/workflows/go.yml/badge.svg)](https://github.com/nornforge/firmware/actions/workflows/go.yml)
[![Latest release](https://img.shields.io/github/v/release/nornforge/firmware)](https://github.com/nornforge/firmware/releases/latest)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

[TinyGo](https://tinygo.org) firmware for a Raspberry Pi Pico that switches the eight relays of the
[Waveshare Pico Relay B](https://www.waveshare.com/wiki/Pico-Relay-B) on request of a host connected over USB.

![Waveshare Pico Relay B](doc/assets/waveshare-pico-relay-b.jpg)

The project is split into two parts:

- this repository holds the firmware running on the Pico
- the [norn](https://github.com/nornforge/norn) repository holds the command line client and REST server
  that talk to it from the host

## Installing a release

### 1. Download and check the image

Download `norn-firmware_pico_<version>.uf2` and `checksums.txt` from the
[latest release](https://github.com/nornforge/firmware/releases/latest) and check them:

```sh
grep -F "  norn-firmware_pico_<version>.uf2" checksums.txt | sha256sum --check -
gh attestation verify norn-firmware_pico_<version>.uf2 --repo nornforge/firmware
```

The first command checks that the download is complete, the second that the image was built by this
repository's release workflow.

### 2. Put the board into flash mode

The first time, the board has to be put into flash mode with its buttons. On the Waveshare Pico Relay B
they sit next to the Pico; in the picture below the boot button is marked **6** and the reset button **7**.

![Buttons of the Waveshare Pico Relay B](https://github.com/nornforge/firmware/assets/6263439/7bc95cdc-e737-4fe6-aa51-592e51f0315b)

1. Connect the Pico to the host with a USB cable.
2. Press and hold the **boot** button (6).
3. Press the **reset** button (7) briefly and release it.
4. Release the **boot** button.

The board now shows up as a USB drive named `RPI-RP2`. If it does not, repeat the steps and make sure
the boot button is still held down while reset is released.

Without the reset button, holding the boot button while plugging in the USB cable has the same effect.

Once this firmware runs on the board, no button is needed anymore: `norn boot` from the
[norn CLI](https://github.com/nornforge/norn) sends it into flash mode.

### 3. Copy the image

Copy the `.uf2` file onto the `RPI-RP2` drive. The board restarts into the new firmware on its own and the
drive disappears. The status LED turns green and the serial port `/dev/ttyACM*` (on Linux) appears.

`norn version` shows the version of the CLI and of the firmware on the board.

## Usage

The firmware shows up on the host as a USB serial port (`/dev/ttyACM*` on Linux).
Use the [norn CLI](https://github.com/nornforge/norn) to switch the relays, read their state
or serve them over a REST API.

See [Intended use](#intended-use) for what the firmware is meant for and
[Relay states](#relay-states) for how the relays behave on start, restart and errors.

### Channels

| Channel | Pico pin |
| ------- | -------- |
| 1       | GP21     |
| 2       | GP20     |
| 3       | GP19     |
| 4       | GP18     |
| 5       | GP17     |
| 6       | GP16     |
| 7       | GP15     |
| 8       | GP14     |

### Protocol

Host and firmware exchange framed JSON messages, implemented in
[`pkg/norn`](https://pkg.go.dev/github.com/nornforge/norn/pkg/norn). A frame looks like this:

```
E979#000000037#e2da8fab{"channel":1,"type":1,"status":false}ba9f13b7
```

| Part        | Content                                                    |
| ----------- | ---------------------------------------------------------- |
| `E979#`     | start marker                                               |
| `000000037` | length of the JSON payload in bytes, nine decimal digits   |
| `#`         | separator                                                  |
| `e2da8fab`  | CRC-32C of the length field, eight hex digits              |
| `{…}`       | JSON payload                                               |
| `ba9f13b7`  | CRC-32C of the JSON payload, eight hex digits              |

The `type` of a command selects what the firmware does:

| Type | Command      | Effect                                                 |
| ---- | ------------ | ------------------------------------------------------ |
| 1    | get          | report the state of `channel`                          |
| 2    | set          | switch `channel` to `status` and report the new state  |
| 3    | version      | report the firmware version in `message`               |
| 4    | bootloader   | reboot into the USB bootloader                         |
| 5    | max channels | report the number of channels in `max_channel`         |
| 6    | brightness   | set the brightness of the status LED, see below        |

Every command except bootloader is answered with a frame of the same layout carrying
`isOk`, `message`, `channel`, `max_channel` and `status`. On an error `isOk` is `false`
and `message` holds the reason.

### Status LED

The RGB LED of the Waveshare Pico Relay B shows what the firmware is doing:

| Color   | Meaning                                                                    |
| ------- | -------------------------------------------------------------------------- |
| green   | the firmware is running and waiting for commands                           |
| blue    | a command was executed, shown for 150 ms                                   |
| red     | a command was answered with an error, shown for one second                 |
| magenta | the bootloader was requested; the LED keeps the color while it is running  |

A new command replaces the color of the previous one, so a red LED turns blue as soon as the
next command succeeds.

The brightness can be changed with command type 6. The value, from 0 (LED off) to 255 (full scale),
is passed in the `channel` field and echoed in the `channel` field of the response. The default is 16.
The setting is not stored: after a restart the default applies again.
With the [norn CLI](https://github.com/nornforge/norn) v0.2.0 or newer, run `norn brightness <0-255>`.

## Intended use

The firmware is meant for switching loads in development and test setups, for example the power supply
of a device under test, from a host that is connected over USB and trusted.

It is not meant for, and must not be relied on in, applications where a relay that switches unexpectedly,
or does not switch, can endanger people or cause damage. It is not a safety component:

- The firmware does not read back the relay contacts. The state it reports is the state it last set.
- The electrical limits are those of the Waveshare Pico Relay B; see its
  [documentation](https://www.waveshare.com/wiki/Pico-Relay-B).
- There is no authentication. Whoever can write to the serial port can switch every relay and send the Pico
  into its bootloader, where new firmware can be installed. Anything that makes the port reachable for others,
  such as the REST server of the norn CLI, has to be protected by the operator.

### Relay states

Off is the state the firmware falls back to. A relay is only on after a `set` command switched it on.

| Situation                                              | Relays                                              |
| ------------------------------------------------------ | --------------------------------------------------- |
| Power-up or reset                                      | all off                                             |
| Bootloader requested by command (`norn boot`)          | all switched off before the bootloader is entered   |
| Firmware stops handling commands for about two seconds | the watchdog restarts the firmware, all off         |
| Invalid or unknown command                             | unchanged, the command is answered with an error    |
| Host program exits, or USB is unplugged while the board stays powered | unchanged                            |
| Power lost                                             | all off                                             |

The relay states are not stored: after every restart all relays are off until they are switched on again.

## Development

The repository comes with a [dev container](.devcontainer/devcontainer.json) that provides Go and TinyGo.
The code imports TinyGo's `machine` package, so build and test it with `tinygo` rather than plain `go`.

```sh
tinygo build -size short -o firmware.uf2 -target=pico ./firmware   # build
tinygo test -v ./...                                                # test
```

A build without further options reports the version `v0.0.0`. To set it, pass

```sh
tinygo build -o firmware.uf2 -target=pico -ldflags="-X 'main.Version=v1.2.3'" ./firmware
```

Flashing from inside the dev container is not supported: copy the resulting `firmware.uf2` onto the Pico
from the host as described in [Installing a release](#installing-a-release).

On a Linux host the Pico's serial port is passed into the dev container as `/dev/ttyACM*`.
The device node keeps the group it has on the host, so access from inside the container may need `sudo`.

### Releasing

Pushing a version tag such as `v1.2.3` runs the tests, builds the firmware with that tag as its version
and publishes a GitHub release with:

- `norn-firmware_pico_<version>.uf2`, the firmware image
- `norn-firmware_pico_<version>.cdx.json`, its software bill of materials (SBOM) in
  [CycloneDX](https://cyclonedx.org) format. It lists the Go modules linked into the image,
  TinyGo and the Go standard library.
- `checksums.txt`, the SHA-256 checksums of both files

Image and SBOM carry a signed [build provenance attestation](https://docs.github.com/en/actions/concepts/security/artifact-attestations).
The changelog is generated from the commit messages, which therefore follow
[Conventional Commits](https://www.conventionalcommits.org).

The same files can be built locally with `scripts/build-release.sh <version>`; they end up in `dist/`.

## License

This software is licensed under the [Apache License Version 2.0](LICENSE)
