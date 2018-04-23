package main

import (
	"github.com/vishvananda/netlink"
	"fmt"
	"github.com/vishvananda/netns"
	"log"
	"syscall"
	"os"
	"github.com/alaypatel07/openvnv/devices"
)

type byBridge []netlink.Link

func (s byBridge) Len() int {
	return len(s)
}

func (s byBridge) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byBridge) Less(i, j int) bool {
	return s[i].Type() == "bridge"
}

func createNamespaceDeleteCallback() (func(namespace devices.Namespace, event devices.NSEvent), *chan bool) {
	doneChannel := make(chan bool)
	callback := func(namespace devices.Namespace, event devices.NSEvent) {
		doneChannel <- true
	}
	return callback, &doneChannel
}

func listenOnLinkMessagesWithExisting(namespace *devices.Namespace, targetNS *netns.NsHandle, consoleDisplay bool) {

	callback, doneChannel := createNamespaceDeleteCallback()
	namespace.OnChange(devices.NSDelete, callback)


	lu := make(chan netlink.LinkUpdate)
	options := netlink.LinkSubscribeOptions{
		Namespace: targetNS,
		ErrorCallback: func(e error) {
			log.Fatalln(e)
		},
		ListExisting: false,
	}
	ldone := make(chan struct{})
	if err := netlink.LinkSubscribeWithOptions(lu, ldone, options); err != nil {
		fmt.Errorf("Link Subscribe error", err)
	}

	for {
		select {
		case update := <-lu:
			if update.Header.Type == syscall.RTM_NEWLINK {
				if update.Change == 0xffffffff {
					namespace.AddL2Device(&update, consoleDisplay)
				} else if update.Change == 0x100 && update.Attrs().Index != 0 {
					namespace.SetMaster(int(update.Attrs().Index), int(update.Attrs().MasterIndex))
				} else if update.Attrs().OperState == netlink.OperUnknown ||
					update.Attrs().OperState == netlink.OperLowerLayerDown ||
					update.Attrs().OperState == netlink.OperUp ||
					update.Attrs().OperState == netlink.OperDown {
					namespace.SetFlags(int(update.Attrs().Index), update.Attrs().Flags, update.Attrs().OperState)
				}
			}
			if update.Header.Type == syscall.RTM_DELLINK {
				if update.Change == 0xffffffff {
					namespace.RemoveDevice(int(update.Index))
				} else if update.Change == 0 && update.Attrs().MasterIndex != 0 {
					namespace.RemoveMaster(int(update.Attrs().Index), int(update.Attrs().MasterIndex))
				}
			}
		case u := <- *doneChannel:
			if u {
				return
			}
		case d := <-ldone:
			fmt.Println("Done ", d)
			os.Exit(1)
		}
	}
}
