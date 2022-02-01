#!/bin/bash

which gosec
if [ $? -ne 0 ]; then
	echo "Downloading gosec tool"
	go install github.com/securego/gosec/v2/cmd/gosec@v2.9.1
fi

time gosec -conf gosec.conf.json ./...
