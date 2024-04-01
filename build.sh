#!/bin/bash

# Usage ./build.sh
docker buildx build --platform linux/amd64,linux/arm64 -t "idanoo/gomatrixcloner:latest" --push .
