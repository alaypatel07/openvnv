#!/usr/bin/env bash

sudo brctl addbr tnet
sudo ip l set tnet up
sudo virsh net-define tnet.xml
sudo virsh net-start tnet
sudo virsh start vm2
sudo virsh attach-interface --domain vm2 --type bridge --source tnet --model virtio --config --live --mac  00:00:00:11:11:11
sudo virsh shutdown vm2
sleep 2
sudo virsh start vm2


read -p "Hit enter to destroy files and network"
sudo virsh detach-interface --domain vm2 --type bridge --mac 00:00:00:11:11:11 --config --live
sudo virsh shutdown vm2
sudo virsh start vm2
sudo ip l set tnet down
sudo virsh net-destroy tnet
sudo virsh net-undefine tnet
sudo brctl delbr tnet

#Expected Output
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
#DEV ID: 3 Event 0 occured which means L2 Device Created
#DEV ID: 3 Event 8 occured which means L2Bridge Created
#DEV ID: 3 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 1 Event 0 occured which means L2 Device Created
#DEV ID: 1 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 2 Event 0 occured which means L2 Device Created
#DEV ID: 2 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 4 Event 0 occured which means L2 Device Created
#DEV ID: 4 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 5 Event 0 occured which means L2 Device Created
#DEV ID: 5 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#
#
#Process existing topology ...Done
#*
#{"name":"virbr0","index":3,"netns":0,"status":0,"master":0,"Ports":{}}
#{"name":"virbr0-nic","index":4,"netns":0,"status":0,"master":0}
#{"name":"ens33","index":2,"netns":0,"status":1,"master":0}
#{"name":"lo","index":1,"netns":0,"status":1,"master":0}
#{"name":"dummy0","index":5,"netns":0,"status":0,"master":0}
#DEV ID: 26 Event 0 occured which means L2 Device Created
#DEV ID: 26 Event 8 occured which means L2Bridge Created
#DEV ID: 26 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 26 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 27 Event 0 occured which means L2 Device Created
#DEV ID: 27 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 27 Event 5 occured which means L2 Device Master Set
#DEV ID: 3 Event 10 occured which means L2Bridge Port Added
#DEV ID: 27 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 28 Event 0 occured which means L2 Device Created
#DEV ID: 28 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 28 Event 5 occured which means L2 Device Master Set
#DEV ID: 26 Event 10 occured which means L2Bridge Port Added
#DEV ID: 26 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 28 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 26 Event 2 occured which means L2 Device Status Flags set. Device UP
#DEV ID: 3 Event 2 occured which means L2 Device Status Flags set. Device UP
#*
#{"name":"lo","index":1,"netns":0,"status":1,"master":0}
#{"name":"vnet0","index":27,"netns":0,"status":1,"master":3}
#{"name":"vnet1","index":28,"netns":0,"status":1,"master":26}
#{"name":"virbr0","index":3,"netns":0,"status":1,"master":0,"Ports":{"0":27}}
#{"name":"virbr0-nic","index":4,"netns":0,"status":0,"master":0}
#{"name":"ens33","index":2,"netns":0,"status":1,"master":0}
#{"name":"dummy0","index":5,"netns":0,"status":0,"master":0}
#{"name":"tnet","index":26,"netns":0,"status":1,"master":0,"Ports":{"0":28}}
#DEV ID: 28 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 28 Event 6 occured which means L2 Device Master Unset
#DEV ID: 26 Event 11 occured which means L2Bridge Port Deleted
#DEV ID: 28 Event 1 occured which means L2 Device Deleted
#DEV ID: 26 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 26 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 26 Event 1 occured which means L2 Device Deleted
#DEV ID: 26 Event 9 occured which means L2Bridge Deleted
#DEV ID: 27 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#DEV ID: 27 Event 6 occured which means L2 Device Master Unset
#DEV ID: 3 Event 11 occured which means L2Bridge Port Deleted
#DEV ID: 27 Event 1 occured which means L2 Device Deleted
#DEV ID: 3 Event 3 occured which means L2 Device Status Flags set. Device DOWN
#*
#{"name":"dummy0","index":5,"netns":0,"status":0,"master":0}
#{"name":"virbr0-nic","index":4,"netns":0,"status":0,"master":0}
#{"name":"virbr0","index":3,"netns":0,"status":0,"master":0,"Ports":{}}
#{"name":"lo","index":1,"netns":0,"status":1,"master":0}
#{"name":"ens33","index":2,"netns":0,"status":1,"master":0}
