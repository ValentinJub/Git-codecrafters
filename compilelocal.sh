#!/bin/sh
cd "$(dirname "$0")" # Ensure compile steps are run within the repository directory
go build -buildvcs="false" -o /tmp/codecrafters-build-git-go ./cmd/mygit/*.go