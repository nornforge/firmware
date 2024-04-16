#!/bin/sh

# print the Go version
go version

# Remove any leftover settings from the TinyGo extension
# This is needed dur to the fact it contains some arbitrary hash value in the GOROOT
# This may change on different iterations of the container
cat ./.vscode/settings.json | jq 'del(."go.toolsEnvVars")' | tee ./.vscode/settings.json >/dev/null

# Install Bash Autocompletion
sudo apt-get update
sudo apt-get install -q -y bash-completion \
                           jq

TINY_GO_VERSION="0.31.2"
# Install Tiny Go
TINY_GO_DEB="tinygo_${TINY_GO_VERSION}_amd64.deb"
wget https://github.com/tinygo-org/tinygo/releases/download/v${TINY_GO_VERSION}/${TINY_GO_DEB}
sudo dpkg -i ${TINY_GO_DEB} && rm ${TINY_GO_DEB}

export PATH=$PATH:/usr/local/bin

go get -u github.com/spf13/cobra@latest
go install github.com/spf13/cobra-cli@latest
go install github.com/sago35/tinygo-autocmpl@latest

echo 'eval "$(tinygo-autocmpl --completion-script-bash)"' >> ~/.bashrc