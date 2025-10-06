#!/usr/bin/env zsh

# exit the script on command errors or unset variables
# http://redsymbol.net/articles/unofficial-bash-strict-mode/
set -euo pipefail
IFS=$'\n\t'

# https://stackoverflow.com/a/246128/295807
script_dir="${0:A:h}"
readonly script_dir
cd "${script_dir}"

# Use a new version of enventory
go install .
export PATH="$HOME/go/bin:$PATH"
echo "Using:"
which fling

export PROMPT='%F{47}$ %f'

vhs < ./demo.tape
