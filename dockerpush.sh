#!/bin/bash

jobID=$1
echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
containerID=`docker ps | awk '{if(NR==2){print $1}}'`
docker commit $containerID copernet/copernicus:$jobID
docker push copernet/copernicus:$jobID
