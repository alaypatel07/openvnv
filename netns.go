package main

import (
	"context"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"
	"github.com/vishvananda/netns"
	"fmt"
	"github.com/docker/docker/api/types/filters"
)

type Namespace struct {
	name        string
	*netns.NsHandle
	doneChannel *chan bool
}

func NewNamespace(name string, ns *netns.NsHandle) Namespace {
	c := make(chan bool)
	return Namespace{
		name,
		ns,
		&c,
	}
}

func SubscribeDockerNetnsUpdate(createUpdate *chan Namespace, destroyUpdate *chan Namespace, errChan chan error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		errChan <- err
		return
	}
	containerList, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		fmt.Println("ERROR: GETTING CONTAINER LIST", err)
		errChan <- err
	}
	for _, container := range containerList {
		nsHandle, err := netns.GetFromDocker(container.ID)
		if err != nil {
			fmt.Println("ERROR: GETTING CONTAINER NETNS", err)
			errChan <- err
		}

		*createUpdate <- NewNamespace(container.ID, &nsHandle)
	}
	opt := types.EventsOptions{
		Since: "",
		Until: "",
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "event", Value: "create"},
			filters.KeyValuePair{Key: "event", Value: "destroy"}),
	}
	updateMessage, errC := cli.Events(ctx, opt)
	for {
		select {
		case u := <-updateMessage:
			if u.Type == "container" && u.Action == "create" {
				nsHandle, err := netns.GetFromDocker(u.ID)
				if err != nil {
					errChan <- err
				}
				*createUpdate <- NewNamespace(u.ID, &nsHandle)
			}
			if u.Type == "container" && u.Action == "destroy" {
				*destroyUpdate <- NewNamespace(u.ID, nil)
			}
		case err := <-errC:
			errChan <- err
		}
	}
}
