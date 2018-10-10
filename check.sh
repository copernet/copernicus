#!/usr/bin/env bash


set -ex

if [ ! -x "$(type -p glide)" ]; then
    exit 1
fi

if [ ! -x "$(type -p gometalinter)" ]; then
    exit 1
fi



find . -name "*.go" -not -path "./vendor/*" -not -path "./git/*" | xargs gofmt -w

linter_targets=$(glide novendor)

test -z "$(gometalinter -j 4 --disable-all \
--enable=gofmt \
--enable=golint \
--enable=gosimple \
--enable=ineffassign \
--enable=vet \
--enable=unconvert \
--exclude='should have comment' \
--exclude='and that stutters;' \
--deadline=10m $linter_targets 2>&1 | grep -v 'ALL_CAPS\|OP_' 2>&1 | tee /dev/stderr)"

go test  -covermode=atomic -coverprofile=coverage.out -race -tags rpctest $linter_targets



