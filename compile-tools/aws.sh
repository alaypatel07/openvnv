#!/usr/bin/env bash

env GOOS=linux GOARCH=amd64 go build -o ./main ./


scp -i  "$HOME/Documents/NCSU/Amazon/Pem/First.pem" ./main ubuntu@ec2-13-58-159-86.us-east-2.compute.amazonaws.com:~/openvnv