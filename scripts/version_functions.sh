#!/bin/bash

# Function to read the SemVer from a file
get_version() {
    local filename=$1

    if [[ -f $filename ]]; then
        version=$(<$filename)
    else
        echo "File $filename not found"
        exit 1
    fi
}

# Function to validate the version form
validate_format() {
    local regex="^v?([0-9]+)\.([0-9]+)\.([0-9]+)(-([0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*))?(\+([0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*))?$"

    if [[ $version =~ $regex ]]; then
        prefix=${BASH_REMATCH[0]}
        major=${BASH_REMATCH[1]}
        minor=${BASH_REMATCH[2]}
        patch=${BASH_REMATCH[3]}
        prerelease=${BASH_REMATCH[5]}
        buildmetadata=${BASH_REMATCH[8]}

        version_matched=true
    else
        version_matched=false
    fi
}

is_prerelease() {
    if [[ -n $prerelease ]]; then
        echo "true"
    else
        echo "false"
    fi
}

print_version() {
    if [[ -z $prerelease ]]; then
        echo "v$major.$minor.$patch"
    else
        if [[ -z $buildmetadata ]]; then
            echo "v$major.$minor.$patch-$prerelease"
        else
            echo "v$major.$minor.$patch-$prerelease+$buildmetadata"
        fi
    fi
}