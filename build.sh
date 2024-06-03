#!/bin/bash

# Build binaries
go mod download
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/linux/amd64/lovo-provisioner .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/linux/arm64/lovo-provisioner .

# Build docker image
docker buildx create --name mybuilder --use
docker buildx build --platform linux/amd64,linux/arm64 -t poneding/lovo-provisioner:latest --push .
# docker buildx rm mybuilder

# Clean up
rm -rf bin