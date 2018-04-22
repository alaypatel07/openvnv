#!/usr/bin/env bash

sudo ip l add du type dummy
sudo ip l add v1 type veth peer name v2
sudo ip l add tr type bridge
sudo ip l set du up
sudo ip l set tr up
sudo ip l set v1 up
sudo ip l set v2 up

read -p "Hit enter to wrap up the test"

sudo ip l del du
sudo ip l del v1
sudo ip l del tr


#Expected Output
#if events are enabled link openvnv -events then
#
#Hello OpenVNV
#Console Display Enabled
#
#Process existing topology ...
#Enter:
#Index Number to look for device state or
#'*' to look for all devices
#'bye' to exit
#'help' to print this message again
#DEV ID: 1 Event 0 occured which means L2 Device Created
#DEV ID: 1 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 2 Event 0 occured which means L2 Device Created
#DEV ID: 2 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 4 Event 0 occured which means L2 Device Created
#DEV ID: 4 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 6 Event 0 occured which means L2 Device Created
#DEV ID: 6 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 160 Event 0 occured which means L2 Device Created
#DEV ID: 160 Event 8 occured which means L2Bridge Created
#DEV ID: 160 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#
#
#Process existing topology ...Done
#*
#{"name":"ens33","index":2,"netns":0,"status":1,"master":0}
#{"name":"lo","index":1,"netns":0,"status":1,"master":0}
#{"name":"dummy0","index":6,"netns":0,"status":0,"master":0}
#{"name":"virbr0-nic","index":4,"netns":0,"status":0,"master":0}
#{"name":"ovs-system","index":160,"netns":0,"status":0,"master":0,"Ports":{}}
#DEV ID: 170 Event 0 occured which means L2 Device Created
#DEV ID: 170 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 171 Event 0 occured which means L2 Device Created
#DEV ID: 171 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 172 Event 0 occured which means L2 Device Created
#DEV ID: 172 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 170 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 172 Event 4 occured which means L2 Device Status Flags set. Device LOWERLAYERDOWN
#DEV ID: 171 Event 4 occured which means L2 Device Status Flags set. Device LOWERLAYERDOWN
#DEV ID: 171 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 172 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 170 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 170 Event 1 occured which means L2 Device Deleted
#DEV ID: 172 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 171 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 172 Event 1 occured which means L2 Device Deleted
#DEV ID: 171 Event 1 occured which means L2 Device Deleted
#*
#{"name":"ovs-system","index":160,"netns":0,"status":0,"master":0,"Ports":{}}
#{"name":"dummy0","index":6,"netns":0,"status":0,"master":0}
#{"name":"lo","index":1,"netns":0,"status":1,"master":0}
#{"name":"ens33","index":2,"netns":0,"status":1,"master":0}
#{"name":"virbr0-nic","index":4,"netns":0,"status":0,"master":0}
