package main

import (
	"github.com/vishvananda/netlink"
	"fmt"
	"os"
)

func listenOnAddressMessages() {
	au := make(chan netlink.AddrUpdate)
	done := make(chan struct{})
	if err := netlink.AddrSubscribe(au, done); err != nil {
		fmt.Errorf("Link Subscribe error", err)
	}
	for {
		select {
		case update := <-au:
			if update.NewAddr {
				addAddress(update)
			} else {
				deleteAddress(update)
			}

		case d := <-done:
			fmt.Println("Done ", d)
			os.Exit(1)
		}
	}
}
func deleteAddress(update netlink.AddrUpdate) {
	fmt.Print("Address deleted ")
	fmt.Printf("%+v\n", update)
}
func addAddress(update netlink.AddrUpdate) {
	fmt.Print("Address added ")
	fmt.Printf("%+v\n", update)
}
