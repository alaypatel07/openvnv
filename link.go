package main

import (
	"github.com/vishvananda/netlink"
	"fmt"
	"sort"
	"github.com/vishvananda/netns"
	"log"
	"syscall"
	"os"
	"runtime"
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

func listenOnLinkMessagesWithExisting(nsHandle *Namespace, consoleDisplay bool) {
	var namespace *devices.Namespace
	runtime.LockOSThread()
	currNs, err := netns.Get()
	if err != nil {
		fmt.Println("ERROR: GETTING CURRENT NS: ", err)
	}
	if currNs.Equal(*nsHandle.NsHandle) {
		namespace = devices.GetDefaultNamespace()
		if namespace == nil {
			fmt.Println("ERROR: TOPOLOGY FOR THIS NAMESPACE IS NOT ASSINED", currNs)
			os.Exit(1)
		}
	}  else {
		if err := netns.Set(*nsHandle.NsHandle); err != nil {
			fmt.Println("ERROR: SETTING NEW NS:", err)
		}
		namespace = devices.GetNamespaceTopology(nsHandle.name)
		if namespace == nil {
			fmt.Println("ERROR: TOPOLOGY FOR THIS NAMESPACE IS NOT ASSINED", currNs)
			os.Exit(1)
		}
	}
	fmt.Println("\nProcessing existing namespace ...")
	links, err := netlink.LinkList()
	if err != nil {
		fmt.Println("Err", err.Error())
	} else {
		sort.Sort(byBridge(links))
		for _, value := range links {
			namespace.AddL2Device(value, consoleDisplay)
		}
	}
	fmt.Println("\n\nProcessing existing namespace ...Done")
	netns.Set(currNs)
	runtime.UnlockOSThread()
	lu := make(chan netlink.LinkUpdate)
	options := netlink.LinkSubscribeOptions{
		Namespace: nsHandle.NsHandle,
		ErrorCallback: func(e error) {
			log.Fatalln(e)
		},
		ListExisting: false,
	}
	ldone := make (chan struct{})
	if err := netlink.LinkSubscribeWithOptions(lu, ldone, options); err != nil {
		fmt.Errorf("Link Subscribe error", err)
	}
	for {
		select {
		case update := <- lu:
			if update.Header.Type == syscall.RTM_NEWLINK {
				if update.Change == 0xffffffff {
					namespace.AddL2Device(&update, consoleDisplay)
				}  else if update.Change == 0x100 && update.Attrs().Index != 0 {
					namespace.SetMaster(int(update.Attrs().Index), int(update.Attrs().MasterIndex))
				} else if update.Attrs().OperState == netlink.OperUnknown 	||
					update.Attrs().OperState == netlink.OperLowerLayerDown 	||
					update.Attrs().OperState == netlink.OperUp 				||
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
		case u := <- *nsHandle.doneChannel:
			if u {
				return
			}
		case d := <- ldone:
			fmt.Println("Done ", d)
			os.Exit(1)
		}
	}
}