package main

import (
	"context"
	"fmt"
	"github.com/alaypatel07/openvnv/devices"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"runtime"
	"sort"
)

var nameLimit = 10

func createDevices(namespace *devices.Namespace, consoleDisplay bool) {
	if consoleDisplay {
		fmt.Println("Processing devices in namespace, ", namespace.Name)
	}
	links, err := netlink.LinkList()
	if err != nil {
		fmt.Println("ERROR: GETTING DEVICES IN DOCKER NS: ", err, namespace)
		return
	}
	sort.Sort(byBridge(links))

	for _, link := range links {
		namespace.AddL2Device(link, consoleDisplay)
		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			fmt.Println("Error in getting addresses", err)
		}
		namespace.AddL3Device(link.Attrs().Index, addrs, consoleDisplay)
	}
	if consoleDisplay {
		fmt.Println("Processing devices in namespace, ", namespace.Name, "...Done")
	}
}

func createExistingNamespaces(consoleDisplay bool) {

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return
	}

	namespace = topology.GetDefaultNamespace()

	createDevices(namespace, consoleDisplay)

	go listenOnLinkMessagesWithExisting(namespace, nil, consoleDisplay)
	go listenOnAddressMessages(namespace, nil)

	containerList, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		fmt.Println("ERROR: GETTING CONTAINER LIST", err)
		return
	}

	for _, container := range containerList {
		runtime.LockOSThread()

		defaultNS, err := netns.Get()
		if err != nil {
			fmt.Println("ERROR: GETTING CURRENT NS: ", err, namespace)
			return
		}

		targetNS, err := netns.GetFromDocker(container.ID[:nameLimit])
		if err != nil {
			fmt.Println("ERROR: GETTING DOCKER NS: ", err, namespace)
			return
		}
		err = netns.Set(targetNS)
		if err != nil {
			fmt.Println("ERROR: SETTING GOROUTINE TO DOCKER NS: ", err, namespace)
			return
		}
		t := topology.CreateNamespace(container.ID[:nameLimit], &targetNS)
		createDevices(t, consoleDisplay)
		err = netns.Set(defaultNS)
		if err != nil {
			fmt.Println("ERROR: SETTING GOROUTINE TO DEFAULT NS: ", err, namespace)
			return
		}

		runtime.UnlockOSThread()
		go listenOnLinkMessagesWithExisting(t, &targetNS, consoleDisplay)
		go listenOnAddressMessages(t, &targetNS)

	}

}

func SubscribeDockerNetnsUpdate(createUpdate *chan string, destroyUpdate *chan string, errChan chan error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		errChan <- err
		return
	}
	opt := types.EventsOptions{
		Since: "",
		Until: "",
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "event", Value: "start"},
			filters.KeyValuePair{Key: "event", Value: "destroy"}),
	}
	updateMessage, errC := cli.Events(ctx, opt)
	for {
		select {
		case u := <-updateMessage:
			if u.Type == "container" && u.Action == "start" {
				*createUpdate <- u.ID[:nameLimit]
			}
			if u.Type == "container" && u.Action == "destroy" {
				*destroyUpdate <- u.ID[:nameLimit]
			}
		case err := <-errC:
			errChan <- err
		}
	}
}
