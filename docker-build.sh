#!/bin/sh

docker build --no-cache -t $DOCKER_ID_USER/sandpiper .
docker push $DOCKER_ID_USER/sandpiper