#!/bin/bash

docker build -t typekit/gladius . && docker run --rm --tty --interactive --volume $(which docker):$(which docker) --volume /var/run/docker.sock:/var/run/docker.sock --publish 8080:8080 typekit/gladius
