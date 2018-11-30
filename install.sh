#!/usr/bin/env bash

go build .
go build ./tools/coperctl

go install
go install ./tools/coperctl
