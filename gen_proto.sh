#!/bin/bash -eu

protodir=../protos

protoc --go_out=protobufs --go-grpc_out=protobufs -I $protodir $protodir/chat.proto