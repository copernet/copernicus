#!/usr/bin/env bash

./check.sh
if [ $? != 0 ]
	exit 1
fi

$GOPATH/bin/goveralls -coverprofile=coverage.out -service=travis-ci -repotoken $1


