#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "$DIR/version_functions.sh"

# Applying functions
get_version $1
validate_format
is_prerelease