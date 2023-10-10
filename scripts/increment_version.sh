#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "$DIR/version_functions.sh"

# Check flag
increment_version() {
    local action=$1
    case $action in
    -M|--major)
        major=$((major+1))
        minor=0
        patch=0
        prerelease=$prerelease
        ;;
    -m|--minor)
        minor=$((minor+1))
        patch=0
        prerelease=$prerelease
        ;;
    -p|--patch)
        patch=$((patch+1))
        prerelease=$prerelease
        ;;
    -r|--prerelease)
        prerelease="prerelease"
        ;;
    -R|--release)
        prerelease=""
        ;;
    *)
        echo "Invalid flag"
        exit 1
        ;;
    esac
    # print new version
    print_version
}

# Main starts here
if [[ -z $2 ]]; then
    echo "No version file supplied"
    exit 1
fi

input_file=$2
temp_file=$(mktemp)

get_version $input_file
validate_format

if [ $version_matched == "true" ]; then
    increment_version $1 $2 > "$temp_file"
    mv "$temp_file" "$input_file"
else
    echo "Version $version is invalid"
    exit 1
fi