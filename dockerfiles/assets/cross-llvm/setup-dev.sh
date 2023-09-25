#!/bin/bash
set -ue

# Go language server for VSCode
go install -v golang.org/x/tools/gopls@latest

# Install Go debugger
go install -v github.com/go-delve/delve/cmd/dlv@latest

# Go interface stub implmeentert
go install -v github.com/josharian/impl@latest

# Install development specific packages and tools
apt install -y zsh curl sudo ccache

# Create the vscode user and group
groupadd -g 1000 vscode
useradd -u 1000 -g 1000 -s /usr/bin/zsh -m vscode 
VSCODE_SUDOERS_FILE_PATH="/etc/sudoers.d/vscode"
echo "vscode ALL=(root) NOPASSWD:ALL" > "$VSCODE_SUDOERS_FILE_PATH"
chmod 440 "$VSCODE_SUDOERS_FILE_PATH"
chown root:root "$VSCODE_SUDOERS_FILE_PATH"

# Create the "/workspaces" folder
WORKSPACES_DIR_PATH="/workspaces"
mkdir -pv "$WORKSPACES_DIR_PATH"
chown vscode:vscode "$WORKSPACES_DIR_PATH"

# Take ownership of the "/go" folder
chown -R vscode:vscode "/go"