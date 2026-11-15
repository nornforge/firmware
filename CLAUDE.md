# CLAUDE.md

TinyGo firmware for a Raspberry Pi Pico (RP2040) that drives the eight relays of the
Waveshare Pico Relay B. A host talks to it over USB serial using the `norn` protocol
(`github.com/nornforge/norn`); the matching CLI lives in the separate `norn` repo.

## Layout

- `firmware/main.go` – serial command loop (parses `norn.Command`, get/set relay channels)
- `firmware/pico.go` – pin mapping for `-target=pico` (build tag `pico`)
- `firmware/xiao-rp2040.go` – pin mapping for `-target=xiao-rp2040` (NOT pin compatible with the Relay B board)
- `channel/` – small wrapper around a `machine.Pin` holding the relay state
- `scripts/build-release.sh` – builds image, SBOM and checksums into `dist/`; used by CI and the release workflow
- `.devcontainer/` – dev container (Go + TinyGo, versions pinned in `postCreateCommand.sh`)

## Commands

```sh
tinygo build -size short -o firmware.uf2 -target=pico ./firmware   # build
tinygo test -v ./...                                                # tests
```

Version is injected via `-ldflags="-X 'main.Version=<tag>'"` (see `scripts/build-release.sh`).
Releases are cut by pushing a `vX.Y.Z` tag; other tags do not trigger the release workflow.
Flashing from inside the dev container is not supported: copy the `.uf2` onto the Pico manually.

## Conventions

- Code imports `machine`, so use `tinygo` (not plain `go build`/`go vet`) for building and testing.
- Keep the TinyGo and Go versions in `.devcontainer/postCreateCommand.sh`, `.devcontainer/devcontainer.json`
  and `.github/workflows/*.yml` in sync.
- The dev container installs the `norn` CLI (version pinned in `.devcontainer/postCreateCommand.sh`); keep it on the
  same version as the `github.com/nornforge/norn` module in `go.mod`. A connected Pico shows up as `/dev/ttyACM0`,
  access needs `sudo` (e.g. `sudo norn version`).
- Actions in the workflows are pinned to commit hashes (version in a trailing comment); Dependabot updates them.
- Commit messages follow Conventional Commits (`feat:`, `fix:`, `ci:` …); the release changelog is generated from them by `chglog`.
