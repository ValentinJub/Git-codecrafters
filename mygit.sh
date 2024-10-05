#!/bin/sh
#
# Use this script to run your program LOCALLY.
#
# Note: Changing this script WILL NOT affect how CodeCrafters runs your program.
#
# Learn more: https://codecrafters.io/program-interface

set -e # Exit early if any commands fail

# Copied from .codecrafters/compile.sh
#
# - Edit this to change how your program compiles locally
# - Edit .codecrafters/compile.sh to change how your program compiles remotely
compile=$(
  cd "$(dirname "$0")" # Ensure compile steps are run within the repository directory
  go build -buildvcs="false" -o /tmp/codecrafters-build-git-go ./cmd/mygit/*.go
)

# Copied from .codecrafters/run.sh
#
# - Edit this to change how your program runs locally
# - Edit .codecrafters/run.sh to change how your program runs remotely
# temp_dir=/Users/valentin/temp/git
# cd $temp_dir 
exec /tmp/codecrafters-build-git-go "$@"

