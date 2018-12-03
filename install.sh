#!/usr/bin/env bash

go install -i -gcflags='all=-N -l' .
go install -i -gcflags='all=-N -l' ./tools/coperctl
