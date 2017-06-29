#!/usr/bin/env bash

set -ex

if [ ! -x "$(type -p glide)" ]; then
    exit 1
fi

if [ ! -x "$(type -p gometalinter)" ]; then
    exit 1
fi

linter_targets=$(glide novendor)

#test -z "$(gometalinter -j 4 --disable-all \
#--enable=gofmt \
#--enable=golint \
#--enable=vet \
#--enable=gosimple \
#--enable=unconvert \
#--deadline=10m $linter_targets 2>&1 | grep -v 'ALL_CAPS\|OP_' 2>&1 | tee /dev/stderr)"



gometalinter -j 4 --disable-all \
--enable=gofmt \
--enable=golint \
--enable=vet \
--enable=gosimple \
--enable=unconvert \
--exclude='should have comment or be unexported' \
--deadline=10m $linter_targets 2>&1 | grep -v 'ALL_CAPS\|OP_' 2>&1 | tee /dev/stderr

env GORACE="halt_on_error=1" go test -race -tags rpctest $linter_targets