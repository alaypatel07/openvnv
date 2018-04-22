#!/usr/bin/env bash

env GOOS=linux GOARCH=amd64 go build -o ./main ./

ip=172.16.80.100

scp -r ./main ./test/ alay@172.16.80.100:/home/alay/linux/project/
#scp -r  alay@172.16.80.100:/home/alay/linux/project/tests
#scp ./test/test2.sh alay@172.16.80.100:/home/alay/linux/project
#scp ./test/test3.sh alay@172.16.80.100:/home/alay/linux/project
#scp ./test/tnet.xml alay@172.16.80.100:/home/alay/linux/project
