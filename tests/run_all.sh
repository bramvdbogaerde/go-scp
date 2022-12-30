#!/usr/bin/env bash

rm tmp/*

echo "Running from $(pwd)"

echo "Starting docker containers"

docker run -d \
   --name go-scp-test \
   -p 2244:22 \
   -e SSH_USERS=bram:1000:1000 \
   -e SSH_ENABLE_PASSWORD_AUTH=true \
   -v $(pwd)/tmp:/data/  \
   -v $(pwd)/data:/input  \
   -v $(pwd)/entrypoint.d/:/etc/entrypoint.d/ \
   panubo/sshd

sleep 5

echo "Running tests"
go test -v 

echo "Tearing down docker containers"
docker stop go-scp-test
docker rm go-scp-test

echo "Cleaning up"
rm tmp/*
