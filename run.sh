#!/bin/bash

cd /root/nats_receive
docker rm -f nats_receive
docker image rm ghcr.io/pgulb/nats_sensors_receive:main
docker pull ghcr.io/pgulb/nats_sensors_receive:main
docker run -d --name nats_receive -v ./credentials.json:/app/credentials.json:ro -v ./nats.creds:/app/nats.creds:ro ghcr.io/pgulb/nats_sensors_receive:main
