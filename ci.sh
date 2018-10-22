#!/usr/bin/env bash

./check.sh
$GOPATH/bin/goveralls -coverprofile=coverage.out -service=travis-ci -repotoken $1


