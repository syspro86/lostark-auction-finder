#!/bin/bash

docker build -f docker/Dockerfile -t loa_build .
docker create --name loa_build loa_build
docker cp loa_build:/tmp/tiny-golang-image/loa.exe .
docker rm loa_build

#GOOS=windows GOARCH=amd64 go build -tags=windows -ldflags -H=windowsgui -o loa.exe ./cmd/loa/
