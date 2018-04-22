#!/usr/bin/env bash

sudo ip l add du type dummy
sudo ip l add v1 type veth peer name v2
sudo ip l add tr type bridge
sudo ip l add tr1 type bridge
sudo ip l set du up
sudo ip l set tr up
sudo ip l set tr1 up
sudo ip l set v1 up
sudo ip l set v2 up
sudo ip l set v1 master tr
sudo ip l set du master tr
sudo ip l set v1 master tr1

read -p "Hit enter to wrap up the test"

sudo ip l del tr
sudo ip l del tr1
sudo ip l del v1
sudo ip l del du

#Hello OpenVNV
#Console Display Enabled
#
#Process existing topology ...
#Enter:
#Index Number to look for device state or
#'*' to look for all devices
#'bye' to exit
#'help' to print this message again
#DEV ID: 3 Event 0 occured which means L2 Device Created
#DEV ID: 3 Event 8 occured which means L2Bridge Created
#DEV ID: 3 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 1 Event 0 occured which means L2 Device Created
#DEV ID: 1 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 2 Event 0 occured which means L2 Device Created
#DEV ID: 2 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 4 Event 0 occured which means L2 Device Created
#DEV ID: 4 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#
#
#Process existing topology ...Done
#*
#{"name":"virbr0","index":3,"netns":0,"status":0,"master":0,"Ports":{}}
#{"name":"lo","index":1,"netns":0,"status":1,"master":0}
#{"name":"ens33","index":2,"netns":0,"status":1,"master":0}
#{"name":"virbr0-nic","index":4,"netns":0,"status":0,"master":0}
#DEV ID: 5 Event 0 occured which means L2 Device Created
#DEV ID: 5 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 6 Event 0 occured which means L2 Device Created
#DEV ID: 6 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 7 Event 0 occured which means L2 Device Created
#DEV ID: 7 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 8 Event 0 occured which means L2 Device Created
#DEV ID: 8 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 9 Event 0 occured which means L2 Device Created
#DEV ID: 9 Event 8 occured which means L2Bridge Created
#DEV ID: 9 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 10 Event 0 occured which means L2 Device Created
#DEV ID: 10 Event 8 occured which means L2Bridge Created
#DEV ID: 10 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 6 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 9 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 10 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 8 Event 4 occured which means L2 Device Status Flags set. Device LOWERLAYERDOWN
#DEV ID: 7 Event 4 occured which means L2 Device Status Flags set. Device LOWERLAYERDOWN
#DEV ID: 8 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 7 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 8 Event 5 occured which means L2 Device Master Set
#DEV ID: 9 Event 10 occured which means L2Bridge Port Added
#DEV ID: 6 Event 5 occured which means L2 Device Master Set
#DEV ID: 9 Event 10 occured which means L2Bridge Port Added
#DEV ID: 9 Event 11 occured which means L2Bridge Port Deleted
#DEV ID: 8 Event 6 occured which means L2 Device Master Unset
#DEV ID: 8 Event 5 occured which means L2 Device Master Set
#DEV ID: 10 Event 10 occured which means L2Bridge Port Added
#DEV ID: 10 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 9 Event 2 occured which means L2 Device Status Flags set. Device UP
#*
#{"name":"dummy0","index":5,"netns":0,"status":0,"master":0}
#{"name":"virbr0","index":3,"netns":0,"status":0,"master":0,"Ports":{}}
#{"name":"virbr0-nic","index":4,"netns":0,"status":0,"master":0}
#{"name":"ens33","index":2,"netns":0,"status":1,"master":0}
#{"name":"lo","index":1,"netns":0,"status":1,"master":0}
#{"name":"tr1","index":10,"netns":0,"status":1,"master":0,"Ports":{"0":8}}
#{"name":"tr","index":9,"netns":0,"status":1,"master":0,"Ports":{"1":6}}
#{"name":"du","index":6,"netns":0,"status":1,"master":9}
#{"name":"v2","index":7,"netns":0,"status":1,"master":0}
#{"name":"v1","index":8,"netns":0,"status":1,"master":10}

