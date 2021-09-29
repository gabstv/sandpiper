#!/bin/sh

GOOS=linux GOARCH=amd64 go build cmd/sandpiper/main.go

docker build -f local.Dockerfile --no-cache -t $DOCKER_ID_USER/sandpiper:apiroutingexp .
docker push $DOCKER_ID_USER/sandpiper:apiroutingexp