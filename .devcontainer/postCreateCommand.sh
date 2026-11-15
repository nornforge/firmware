#!/bin/sh
set -e

TINY_GO_VERSION="0.44.0"

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
TINY_GO_DEB="tinygo_${TINY_GO_VERSION}_${ARCH}.deb"
DL_DIR=$(mktemp -d)
wget -q -O "${DL_DIR}/${TINY_GO_DEB}" \
    "https://github.com/tinygo-org/tinygo/releases/download/v${TINY_GO_VERSION}/${TINY_GO_DEB}"
sudo dpkg -i "${DL_DIR}/${TINY_GO_DEB}"
rm -rf "${DL_DIR}"
tinygo version

export PATH=$PATH:/usr/local/bin

go install github.com/spf13/cobra-cli@latest
go install github.com/sago35/tinygo-autocmpl@latest

echo 'eval "$(tinygo-autocmpl --completion-script-bash)"' >> ~/.bashrc
