#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "$DIR/version_functions.sh"

# Applying functions
get_version $1
validate_format

if [ $version_matched == "true" ]; then
    print_version
else
    echo "Version $version is invalid"
    exit 1
fi