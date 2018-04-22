#!/usr/bin/env bash

env GOOS=linux GOARCH=amd64 go build -o ./main ./

ip=alay@172.16.80.161
path=/home/alay/golang/src/github.com/alaypatel07/openvnv/
echo $ip:$path
scp -r ./* $ip:$path
