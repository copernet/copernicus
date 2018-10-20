#!/usr/bin/env bash

./check.sh
ls
$GOPATH/bin/goveralls -coverprofile=coverage.out -service=travis-ci -repotoken $1
sleep 5


