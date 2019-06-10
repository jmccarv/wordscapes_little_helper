#!/bin/sh

# Allow this to work in your go workspace so you can
# use go get
export GO111MODULE=on

fatal() {
    echo "ERROR: $*"
    exit 1
}

cd bin || mkdir bin || fatal "No bin/ directory"
for cmd in ../cmd/*; do go build $cmd; done
