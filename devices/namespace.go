package devices

import (
	"github.com/vishvananda/netlink"
	"fmt"
	"net"
	"errors"
)

type NSEvent int

const (
	NSCreate            NSEvent = iota
	NSDelete
	NSTypeChange
	NSConnect
)

var NSEventStrings = []string{
	"NSCreate",
	"NSDelete",
	"NSTypeChange",
	"NSConnect",
}

func (e NSEvent) String() string {
	for i, str := range NSEventStrings {
		if i == int(e) {
			return str
		}
	}
	return ""
}

var defaultSubscriber []func(Namespace, NSEvent)

func SubscribeAllNamespaceEvents(callback func(Namespace, NSEvent)) {
	defaultSubscriber = append(defaultSubscriber, callback)
}

type Namespace struct {
	Name        string
	Type        string
	L2Devices   map[int]LinkUpdateReceiver
	L3Devices   map[int]LinkAddrUpdateReceiver
	Connections map[string]Namespace
	onchange    map[NSEvent][]func(namespace Namespace, change NSEvent)
}

func (n Namespace) OnChange(event NSEvent, callback func(Namespace, NSEvent)) error {
	if int(event) >= len(NSEventStrings) || int(event) < 0 {
		return errors.New("Namespace OnChange: NSEvent unrecognized")
	}
	n.onchange[event] = append(n.onchange[event], callback)
	return nil
}

func NewNamespace(name string) Namespace {
	n := Namespace{
		Name:        name,
		L2Devices:   make(map[int]LinkUpdateReceiver),
		L3Devices:   make(map[int]LinkAddrUpdateReceiver),
		Connections: make(map[string]Namespace),
		onchange:    make(map[NSEvent][]func(Namespace, NSEvent)),
	}
	for index, _ := range NSEventStrings {
		for _, defaultCallback := range defaultSubscriber {
			if err := n.OnChange(NSEvent(index), defaultCallback); err != nil {
				fmt.Println("ERROR: ASSIGNING ONCHANGE", err)
			}
		}
	}
	n.fire(NSCreate)
	n.SetType("bridged")
	return n
}

func (n Namespace) AddL2Device(update netlink.Link, consoleDisplay bool) {
	index := update.Attrs().Index
	if _, ok := n.L2Devices[index]; ok {
		fmt.Println("ADDL2DEVICE: Device", index, "Already Exist")
		return
	}
	var lu LinkUpdateReceiver
	switch update.Type() {
	case "bridge":
		l := NewL2Bridge(update, n.Name,consoleDisplay)
		l.SetFlags(update.Attrs().Flags, update.Attrs().OperState)
		lu = l
		n.L2Devices[update.Attrs().Index] = lu
		go lu.ReceiveLinkUpdate()
	default:
		l := NewL2Device(update, n.Name, consoleDisplay)
		l.SetFlags(update.Attrs().Flags, update.Attrs().OperState)
		lu = l
		go lu.ReceiveLinkUpdate()
		n.L2Devices[update.Attrs().Index] = lu
		n.SetMaster(update.Attrs().Index, update.Attrs().MasterIndex)
	}
}

func (n Namespace) AddL3Device(index int, receiver LinkAddrUpdateReceiver) {
	n.L3Devices[index] = receiver
}

func (n Namespace) RemoveDevice(index int) {
	if dev, ok := n.L2Devices[index]; ok {
		if d, ok := dev.(*L2Device); ok {
			d.DeleteDevice()
		} else if d, ok := dev.(*L2Bridge); ok {
			d.DeleteDevice()
		}
		delete(n.L2Devices, index)
	}
	if _, ok := n.L3Devices[index]; ok {
		delete(n.L3Devices, index)
	}
}

func (n Namespace) SetFlags(index int, f net.Flags, o netlink.LinkOperState) {
	if d, ok := n.L2Devices[index]; ok {
		e := newL2DeviceFlagsEvent(f, o)
		*(d.l2EventChannel().flagsChannel) <- e

	}
	//TODO
	//if d, ok := t.L3Devices[index]; ok {
	//}
}

func (n Namespace) SetMaster(devIndex int, masterIndex int) {
	if d, ok := n.L2Devices[devIndex]; ok {
		if m, ok := n.L2Devices[masterIndex]; ok {
			if masterIndex == 0 || masterIndex == d.l2EventChannel().master {
				return
			}
			e := newL2DeviceMasterEvent(devIndex, masterIndex)
			*(d.l2EventChannel().masterChannel) <- e
			*(m.l2EventChannel()).masterChannel <- e
		}
	}

}

func (n Namespace) RemoveMaster(devIndex int, masterIndex int) {
	if d, ok := n.L2Devices[devIndex]; ok {
		if m, ok := n.L2Devices[masterIndex]; ok {
			e := newL2DeviceMasterEvent(devIndex, 0)
			*(d.l2EventChannel().masterChannel) <- e
			*(m.l2EventChannel()).masterChannel <- e
		}
	}
}

func (n Namespace) Dump(index int) {
	if d, ok := n.L2Devices[index]; ok {
		*d.l2EventChannel().dump <- true
	} else {
		fmt.Println("No device with the index")
	}
}

func (n Namespace) DumpAll() {
	for _, value := range n.L2Devices {
		*value.l2EventChannel().dump <- true
		_ = <-*value.l2EventChannel().dump
	}
}
func (n Namespace) fire(event NSEvent) {
	for _, callback := range n.onchange[event] {
		callback(n, event)
	}
}

func (n Namespace) SetType(s string) {
	if n.Type == s {
		return
	}
	n.Type = s
	n.fire(NSTypeChange)
}

func (n Namespace) Delete() {
	n.fire(NSDelete)
}
