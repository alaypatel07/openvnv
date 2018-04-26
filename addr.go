package main

import (
	"fmt"
	"github.com/alaypatel07/openvnv/devices"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"os"
)

func listenOnAddressMessages(namespace *devices.Namespace, targetNs *netns.NsHandle) {

	au := make(chan netlink.AddrUpdate)
	done := make(chan struct{})
	if targetNs == nil {
		if err := netlink.AddrSubscribe(au, done); err != nil {
			fmt.Errorf("Link Subscribe error", err)
		}
	} else {
		if err := netlink.AddrSubscribeAt(*targetNs, au, done); err != nil {
			fmt.Errorf("Link Subscribe error", err)
		}
	}

	for {
		select {
		case update := <-au:
			if update.NewAddr {
				namespace.AddL3Addr(update.LinkIndex, &update.LinkAddress)
			} else {
				namespace.RemoveL3Addr(update.LinkIndex, &update.LinkAddress)
			}

		case d := <-done:
			fmt.Println("Done ", d)
			os.Exit(1)
		}
	}
}
