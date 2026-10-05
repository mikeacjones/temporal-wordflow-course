#!/usr/bin/env bash
set -Eeuo pipefail
cd "$(dirname "$0")/.."

# The course site shows these steps in a "Setting up" banner.
step="Starting"
status() { printf '{"step": "%s", "done": %s, "failed": %s}\n' "$step" "$1" "$2" > .setup-status.json; }
next() { step="$1"; status false false; echo "==> $step"; }
trap 'status false true' ERR

next "Starting the course site"
bash .devcontainer/start-course.sh

# Temporal CLI: runs the local Temporal Service and Temporal UI.
next "Installing the Temporal CLI"
curl -sSf https://temporal.download/cli.sh | sh
sudo ln -sf "$HOME/.temporalio/bin/temporal" /usr/local/bin/temporal
temporal --version

next "Downloading Go modules"
go mod download

next "Building the project"
go build ./...

step="Ready"
status true false
