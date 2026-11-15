#!/usr/bin/env bash
# Builds the release artifacts for the Raspberry Pi Pico:
#
#   norn-firmware_pico_<version>.uf2       the firmware image
#   norn-firmware_pico_<version>.cdx.json  its SBOM (CycloneDX 1.5)
#   checksums.txt                          SHA-256 checksums of both
#
# usage: scripts/build-release.sh <version> [output directory, default: dist]
set -euo pipefail

VERSION="${1:?usage: $0 <version> [output directory]}"
DIST="${2:-dist}"
TARGET="pico"
IMAGE="norn-firmware_${TARGET}_${VERSION}.uf2"
SBOM="norn-firmware_${TARGET}_${VERSION}.cdx.json"

cd "$(dirname "$0")/.."
mkdir -p "${DIST}"

tinygo build -size full -o "${DIST}/${IMAGE}" -target="${TARGET}" \
    -ldflags="-X 'main.Version=${VERSION}'" ./firmware

# The SBOM lists what ends up in the image: the Go modules TinyGo links into
# it, plus TinyGo itself and the Go standard library it compiles from. TinyGo
# stands for everything it bundles (runtime, machine package, compiler-rt,
# picolibc and the second stage bootloader).
TINYGO_VERSION=$(tinygo version | sed -n 's/^tinygo version \([^ ]*\).*/\1/p')
GO_VERSION=$(tinygo version | sed -n 's/.*using go version go\([^ ]*\).*/\1/p')
: "${TINYGO_VERSION:?unable to detect the TinyGo version}"
: "${GO_VERSION:?unable to detect the Go version}"

IMAGE_SHA256=$(sha256sum "${DIST}/${IMAGE}" | cut -d' ' -f1)
TIMESTAMP=$(git log -1 --format=%cI 2>/dev/null || date -u +%Y-%m-%dT%H:%M:%SZ)

tinygo list -target="${TARGET}" -deps -json ./firmware | jq -s \
    --arg version "${VERSION}" \
    --arg target "${TARGET}" \
    --arg image "${IMAGE}" \
    --arg sha256 "${IMAGE_SHA256}" \
    --arg tinygo "${TINYGO_VERSION}" \
    --arg go "${GO_VERSION}" \
    --arg timestamp "${TIMESTAMP}" \
    --arg serial "urn:uuid:$(cat /proc/sys/kernel/random/uuid)" '
    ([.[] | .Module | select(. != null and (.Main | not))] | unique_by(.Path)
        | map({
            type: "library",
            "bom-ref": "pkg:golang/\(.Path)@\(.Version)",
            name: .Path,
            version: .Version,
            purl: "pkg:golang/\(.Path)@\(.Version)"
        })) as $modules
    | [
        {
            type: "framework",
            "bom-ref": "pkg:github/tinygo-org/tinygo@v\($tinygo)",
            name: "tinygo",
            version: $tinygo,
            description: "Compiler, runtime and machine package, including the C libraries and second stage bootloader it bundles",
            licenses: [{license: {id: "BSD-3-Clause"}}],
            purl: "pkg:github/tinygo-org/tinygo@v\($tinygo)"
        },
        {
            type: "library",
            "bom-ref": "pkg:golang/stdlib@\($go)",
            name: "stdlib",
            version: $go,
            description: "Go standard library",
            licenses: [{license: {id: "BSD-3-Clause"}}],
            purl: "pkg:golang/stdlib@\($go)"
        }
    ] as $toolchain
    | "pkg:github/nornforge/firmware@\($version)" as $root
    | {
        bomFormat: "CycloneDX",
        specVersion: "1.5",
        serialNumber: $serial,
        version: 1,
        metadata: {
            timestamp: $timestamp,
            tools: [{vendor: "nornforge", name: "build-release.sh"}],
            component: {
                type: "firmware",
                "bom-ref": $root,
                name: "norn-firmware",
                version: $version,
                description: "Firmware for the Waveshare Pico Relay B (\($image))",
                licenses: [{license: {id: "Apache-2.0"}}],
                purl: $root,
                hashes: [{alg: "SHA-256", content: $sha256}],
                properties: [{name: "nornforge:target", value: $target}]
            }
        },
        components: ($modules + $toolchain),
        dependencies: (
            [{ref: $root, dependsOn: [($modules + $toolchain)[]."bom-ref"]}]
            + [($modules + $toolchain)[] | {ref: ."bom-ref", dependsOn: []}]
        )
    }' > "${DIST}/${SBOM}"

(cd "${DIST}" && sha256sum "${IMAGE}" "${SBOM}" > checksums.txt)

echo "Release artifacts in ${DIST}:"
ls -l "${DIST}"
