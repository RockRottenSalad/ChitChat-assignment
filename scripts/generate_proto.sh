#!/usr/bin/env bash

# Assuming script is placed inside a directory whose parent is the project root
PROJECT_ROOT=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd | sed -r "s./[^/]+$..")

PROTO_DIR=proto

protoc -I ${PROJECT_ROOT} \
    --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    ${PROTO_DIR}/chitchat.proto

