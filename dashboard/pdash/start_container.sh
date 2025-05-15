#!/bin/bash

docker build -t proxy-dash .    

# stop running container
docker stop proxyDash

# remove running container
docker rm proxyDash

# run new container
docker run --name proxyDash -e API_BASE_URL=http://host.docker.internal:3000   -p:8501:8501  -d proxy-dash:latest
