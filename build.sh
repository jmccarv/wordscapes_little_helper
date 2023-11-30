#!/bin/sh

# Allow this to work in your go workspace so you can
# use go get
export GO111MODULE=on

fatal() {
    echo "ERROR: $*"
    exit 1
}

test -d bin || mkdir bin || fatal "Failed to create directory 'bin'"
cd bin || fatal "Faled to switch to directory 'bin'"

for cmd in ../cmd/*; do go build $cmd; done
