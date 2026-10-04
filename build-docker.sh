#!/bin/bash
VERSION=$(git describe --tags --always)

if [ -z "$VERSION" ]; then
    echo "No version found"
    exit 1
fi

if [ -z "$DOCKER_NAME" ]; then
    echo "DOCKER_NAME not set"
    exit 1
fi

docker build -t "$DOCKER_NAME:latest" -t "$DOCKER_NAME:$VERSION" .
docker push "$DOCKER_NAME:latest"
docker push "$DOCKER_NAME:$VERSION"

