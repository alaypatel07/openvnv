#!/usr/bin/env bash

sudo ip l add du type dummy
sudo ip l add v1 type veth peer name v2
sudo ip l add tr type bridge
sudo ip l set du up
sudo ip l set tr up
sudo ip l set v1 up
sudo ip l set v2 up
sudo ip l set v1 master tr
sudo ip l set du master tr
sudo ip netns add trial
sudo ip l set v1 netns trial


read -p "Hit enter to wrap up the test"

sudo ip netns del trial
sudo ip l del du
#sudo ip l del v2
sudo ip l del tr
