#!/bin/bash

which gosec
if [ $? -ne 0 ]; then
	echo "Downloading gosec tool"
	go get -u github.com/securego/gosec/v2/cmd/gosec
fi

time GO111MODULE=off gosec -conf gosec.conf.json ./...
