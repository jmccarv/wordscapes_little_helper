#!/bin/sh

fatal() {
    echo "ERROR: $*"
    exit 1
}

cd bin || fatal "No bin/ directory"
for cmd in ../cmd/*; do go build $cmd; done
