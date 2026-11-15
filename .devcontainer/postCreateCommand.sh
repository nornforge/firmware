#!/bin/sh
set -e

TINY_GO_VERSION="0.44.0"
# SHA-256 of tinygo_${TINY_GO_VERSION}_<arch>.deb, update together with the version.
# TinyGo publishes no checksum file; take the values from the release assets:
#   gh api repos/tinygo-org/tinygo/releases/tags/v${TINY_GO_VERSION} -q '.assets[] | "\(.name) \(.digest)"'
TINY_GO_SHA256_AMD64="8810b20d88236a1353e35641a6a56df1720a7db97006aacc2c6656a3bb090b74"
TINY_GO_SHA256_ARM64="c2addff1430c7066fc11900e93ee8d6dd14fce343a7b0e1d72da0509a58766ff"

NORN_VERSION="0.1.4"
# SHA-256 of norn_${NORN_VERSION}_linux_<arch>.tar.gz, update together with the version.
# The values are listed in norn_${NORN_VERSION}_checksums.txt of the release.
NORN_SHA256_AMD64="f53ee78908b13b147df7471e867436c8470eeadb81692f1bab5663b4a812c099"
NORN_SHA256_ARM64="e2bc05691b4cb101550540aa82ce0119cd112a6633bdaa227413f8c5228dc1fa"

# The volume holding the GitHub CLI login is created owned by root
sudo chown -R "$(id -u):$(id -g)" "${HOME}/.config"

# print the Go version
go version

# Install Bash Autocompletion and helpers
sudo apt-get update
sudo apt-get install -q -y bash-completion \
                           jq \
                           wget

# Remove any leftover settings from the TinyGo extension
# This is needed due to the fact it contains some arbitrary hash value in the GOROOT
# This may change on different iterations of the container
if [ -f ./.vscode/settings.json ]; then
    tmp=$(mktemp)
    jq 'del(."go.toolsEnvVars")' ./.vscode/settings.json > "$tmp" && mv "$tmp" ./.vscode/settings.json
fi

# Install TinyGo
ARCH=$(dpkg --print-architecture)
case "${ARCH}" in
    amd64) TINY_GO_SHA256="${TINY_GO_SHA256_AMD64}"; NORN_SHA256="${NORN_SHA256_AMD64}" ;;
    arm64) TINY_GO_SHA256="${TINY_GO_SHA256_ARM64}"; NORN_SHA256="${NORN_SHA256_ARM64}" ;;
    *) echo "No checksums known for architecture ${ARCH}" >&2; exit 1 ;;
esac
TINY_GO_DEB="tinygo_${TINY_GO_VERSION}_${ARCH}.deb"
DL_DIR=$(mktemp -d)
wget -q -O "${DL_DIR}/${TINY_GO_DEB}" \
    "https://github.com/tinygo-org/tinygo/releases/download/v${TINY_GO_VERSION}/${TINY_GO_DEB}"
echo "${TINY_GO_SHA256}  ${DL_DIR}/${TINY_GO_DEB}" | sha256sum -c -
sudo dpkg -i "${DL_DIR}/${TINY_GO_DEB}"
tinygo version

# Install the norn CLI, the host side counterpart of the firmware
NORN_TAR="norn_${NORN_VERSION}_linux_${ARCH}.tar.gz"
wget -q -O "${DL_DIR}/${NORN_TAR}" \
    "https://github.com/nornforge/norn/releases/download/v${NORN_VERSION}/${NORN_TAR}"
echo "${NORN_SHA256}  ${DL_DIR}/${NORN_TAR}" | sha256sum -c -
tar -xzf "${DL_DIR}/${NORN_TAR}" -C "${DL_DIR}" norn
sudo install -m 0755 "${DL_DIR}/norn" /usr/local/bin/norn
/usr/local/bin/norn completion bash > "${DL_DIR}/norn.bash"
sudo install -m 0644 "${DL_DIR}/norn.bash" /etc/bash_completion.d/norn

rm -rf "${DL_DIR}"

export PATH=$PATH:/usr/local/bin

go install github.com/sago35/tinygo-autocmpl@latest

COMPLETION='eval "$(tinygo-autocmpl --completion-script-bash)"'
grep -qxF "${COMPLETION}" ~/.bashrc || echo "${COMPLETION}" >> ~/.bashrc
