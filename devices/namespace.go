package devices

import (
	"errors"
	"fmt"
	"net"

	"strconv"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type NSEvent int

const (
	NSCreate NSEvent = iota
	NSDelete
	NSTypeChange
	NSConnect
	NSDisconnect
	NSRouteAdd
	NSRouteDelete
)

var NSEventStrings = []string{
	"NSCreate",
	"NSDelete",
	"NSTypeChange",
	"NSConnect",
	"NSDisconnect",
	"NSRouteAdd",
	"NSRouteDelete",
}

func (e NSEvent) String() string {
	for i, str := range NSEventStrings {
		if i == int(e) {
			return str
		}
	}
	return ""
}

var defaultNsSubscriber []func(*Namespace, NSEvent)

func SubscribeAllNamespaceEvents(callback func(*Namespace, NSEvent)) {
	defaultNsSubscriber = append(defaultNsSubscriber, callback)
}

type Namespace struct {
	Name           string
	Type           string
	nsHandle       *netns.NsHandle
	L2Devices      map[int]LinkUpdateReceiver
	L3Devices      map[int]LinkAddrUpdateReceiver
	Connections    map[string]string
	onchange       map[NSEvent][]func(namespace *Namespace, change NSEvent)
	Routes         []netlink.Route
	topology       *Topology
	peeringChannel *chan PeerEvent
	Event          string `json:"event"`
}

func (n *Namespace) OnChange(event NSEvent, callback func(*Namespace, NSEvent)) error {
	if int(event) >= len(NSEventStrings) || int(event) < 0 {
		return errors.New("Namespace OnChange: NSEvent unrecognized")
	}
	n.onchange[event] = append(n.onchange[event], callback)
	return nil
}

func NewNamespace(name string, t *Topology, targetNs *netns.NsHandle) Namespace {
	p := make(chan PeerEvent)
	r := make([]netlink.Route, 0)
	n := Namespace{
		Name:           name,
		L2Devices:      make(map[int]LinkUpdateReceiver),
		L3Devices:      make(map[int]LinkAddrUpdateReceiver),
		Connections:    make(map[string]string),
		nsHandle:       targetNs,
		topology:       t,
		onchange:       make(map[NSEvent][]func(*Namespace, NSEvent)),
		Routes:         r,
		peeringChannel: &p,
	}
	for index, _ := range NSEventStrings {
		for _, defaultCallback := range defaultNsSubscriber {
			if err := n.OnChange(NSEvent(index), defaultCallback); err != nil {
				fmt.Println("ERROR: ASSIGNING ONCHANGE", err)
			}
		}
	}
	n.fire(NSCreate)
	n.SetType("bridged")
	return n
}

func (n *Namespace) AddL2Device(update netlink.Link, consoleDisplay bool) {
	index := update.Attrs().Index
	if _, ok := n.L2Devices[index]; ok {
		fmt.Println("ADDL2DEVICE: Device", index, "Already Exist")
		return
	}
	var lu LinkUpdateReceiver
	switch update.Type() {
	case "bridge":
		l := NewL2Bridge(update, n.topology, n.Name, consoleDisplay)
		l.SetFlags(update.Attrs().Flags, update.Attrs().OperState)
		lu = l
		n.L2Devices[update.Attrs().Index] = lu
		go lu.ReceiveLinkUpdate()
	case "veth":
		v, err := NewVeth(update, n.topology, n.Name, consoleDisplay)
		if err != nil {
			fmt.Println("ERROR IN GETTING NEW VETH INTERFACE", err)
		}
		v.SetFlags(update.Attrs().Flags, update.Attrs().OperState)
		lu = v
		n.L2Devices[update.Attrs().Index] = lu
		go lu.ReceiveLinkUpdate()
		n.SetMaster(update.Attrs().Index, update.Attrs().MasterIndex)
	default:
		l := NewL2Device(update, n.topology, n.Name, consoleDisplay)
		l.SetFlags(update.Attrs().Flags, update.Attrs().OperState)
		lu = l
		go lu.ReceiveLinkUpdate()
		n.L2Devices[update.Attrs().Index] = lu
		n.SetMaster(update.Attrs().Index, update.Attrs().MasterIndex)
	}
}

func (n *Namespace) AddL3Device(index int, addrs []netlink.Addr, consoleDisplay bool) {
	ipAddrs := make([]*net.IPNet, 0)
	for _, a := range addrs {
		ipAddrs = append(ipAddrs, a.IPNet)
	}
	if d, ok := n.L2Devices[index]; ok {
		l3dev := NewL3Device(index, n.Name, d, ipAddrs, consoleDisplay)
		n.L3Devices[index] = l3dev
		n.SetType("network")
	}
}

func (n *Namespace) AddL3Addr(index int, addr *net.IPNet) {
	if d, ok := n.L3Devices[index]; ok {
		*d.L3EventChannel().addAddrChannel <- addr
		n.SetType("network")
	}
}

func (n *Namespace) RemoveL3Addr(index int, addr *net.IPNet) {
	if d, ok := n.L3Devices[index]; ok {
		*d.L3EventChannel().removeAddrChannel <- addr
	}
}

func (n *Namespace) RemoveDevice(index int) {
	if dev, ok := n.L3Devices[index]; ok {
		*dev.L3EventChannel().doneChannel <- true
		<-*dev.L3EventChannel().doneChannel
		delete(n.L3Devices, index)
	}
	if dev, ok := n.L2Devices[index]; ok {
		if d, ok := dev.(*L2Device); ok {
			d.DeleteDevice()
		} else if d, ok := dev.(*L2Bridge); ok {
			d.DeleteDevice()
		} else if d, ok := dev.(*Veth); ok {
			d.DeleteDevice()
		}
		delete(n.L2Devices, index)
	}
}

func (n *Namespace) SetFlags(index int, f net.Flags, o netlink.LinkOperState) {
	if d, ok := n.L2Devices[index]; ok {
		e := newL2DeviceFlagsEvent(f, o)
		*(d.L2EventChannel().flagsChannel) <- e

	}
	//TODO
	//if d, ok := topology.L3Devices[index]; ok {
	//}
}

func (n *Namespace) SetMaster(devIndex int, masterIndex int) {
	if d, ok := n.L2Devices[devIndex]; ok {
		if m, ok := n.L2Devices[masterIndex]; ok {
			if masterIndex == 0 || masterIndex == d.L2EventChannel().Master {
				return
			}
			e := newL2DeviceMasterEvent(devIndex, masterIndex)
			*(d.L2EventChannel().masterChannel) <- e
			*(m.L2EventChannel()).masterChannel <- e
		}
	}

}

func (n *Namespace) RemoveMaster(devIndex int, masterIndex int) {
	if d, ok := n.L2Devices[devIndex]; ok {
		if m, ok := n.L2Devices[masterIndex]; ok {
			e := newL2DeviceMasterEvent(devIndex, 0)
			*(d.L2EventChannel().masterChannel) <- e
			*(m.L2EventChannel()).masterChannel <- e
		}
	}
}

func (n *Namespace) Dump(index int) {
	if d, ok := n.L2Devices[index]; ok {
		*d.L2EventChannel().dump <- true
		<-*d.L2EventChannel().dump
		if l3d, ok := n.L3Devices[index]; ok {
			*l3d.L3EventChannel().dumpChannel <- true
			<-*l3d.L3EventChannel().dumpChannel
		}
	} else {
		fmt.Println("No device with the index")
	}
}

func (n *Namespace) DumpAll() {
	for _, value := range n.L3Devices {
		*value.L3EventChannel().dumpChannel <- true
		_ = <-*value.L3EventChannel().dumpChannel
	}
	for index, _ := range n.L2Devices {
		if dev, ok := n.L2Devices[index]; !ok {
			*dev.L2EventChannel().dump <- true
			<-*dev.L2EventChannel().dump
		}
	}
}

func (n *Namespace) fire(event NSEvent) {
	for _, callback := range n.onchange[event] {
		callback(n, event)
	}
}

func (n *Namespace) ChangeDeviceName(devIndex int, newName string) {
	if d, ok := n.L2Devices[devIndex]; ok {
		*d.L2EventChannel().nameChannel <- newName
	}
}

func getNSIndex(namespace string, index int) string {
	return namespace + ":" + strconv.Itoa(index)
}

func (n *Namespace) SendVPCreateEvent(event PeerEvent) {
	fmt.Printf("\ncreate event\n%+v%+v\n", event.Veth, event.Event)
	//fmt.Printf("CREATE %+v %+v\n", *event.Veth, event.Event)
	if e, ok := n.topology.GetEvent(event); ok {
		if e.Event == VPDelete {
			fmt.Printf("create previously own delete event\n%+v\n%+v", e.Veth, e.Event)
			n.topology.RemoveFromBuffer(event.GetIndex())
			n.topology.Disconnect(getNSIndex(e.PeerNamespace, e.PeerIndex), getNSIndex(e.Namespace, e.Index))
			event.Pair(e.PeerIndex, e.PeerName, e.PeerNamespace)
			//e.Pair(event.PeerIndex, event.PeerName, event.PeerNamespace)
			n.topology.Get(e.PeerNamespace).SetVethPeer(e.PeerIndex, event.Index, event.Name, event.Namespace)
			n.topology.Connect(getNSIndex(e.PeerNamespace, event.PeerIndex),
				getNSIndex(event.Namespace, event.Index))
			//n.topology.Connect(e.PeerNamespace + ":"+ strings.itoa(e.PeerIndex), event.Namespace + ":" + strings.itoa(event.Index))
			return
		}
	} else if e, ok := n.topology.GetPeerEvent(event); ok {
		//fmt.Println("Create Got peer event\n")
		fmt.Printf("create previously peer create event\n%+v\n%+v", e.Veth, e.Event)
		if e.Event == VPCreate {
			n.topology.RemoveFromBuffer(event.GetPeerIndex())
			event.Pair(e.Index, e.Name, e.Namespace)
			e.Pair(event.Index, event.Name, event.Namespace)
			n.topology.Connect(getNSIndex(event.Namespace, event.Index), getNSIndex(e.Namespace, e.Index))
			return
		}
	}
	n.topology.AddToBuffer(event)
}

func (n *Namespace) SendVPDeleteEvent(event PeerEvent) {

	fmt.Printf("\ndelete event\n%+v%+v\n", event.Veth, event.Event)
	//if e, ok := n.topology.buffer[event.GetIndex()]; ok {
	//	//n.topology.Connect()
	//}
	//fmt.Printf("Delete %+v %+v\n", *event.Veth, event.Event)
	if e, ok := n.topology.GetEvent(event); ok {
		//fmt.Println("Got own event\n")
		if e.Event == VPCreate {
			fmt.Printf("delete previously own create event\n%+v\n%+v", e.Veth, e.Event)
			n.topology.RemoveFromBuffer(event.GetIndex())
			e.Pair(event.PeerIndex, event.PeerName, event.PeerNamespace)
			n.topology.Get(e.PeerNamespace).SetVethPeer(event.PeerIndex, e.Index, e.Name, e.Namespace)
			n.topology.Disconnect(getNSIndex(event.Namespace, event.Index), getNSIndex(event.PeerNamespace, event.PeerIndex))
			n.topology.Connect(getNSIndex(e.Namespace, e.Index), getNSIndex(event.PeerNamespace, event.PeerIndex))
			return
		}
	} else if e, ok := n.topology.GetPeerEvent(event); ok {
		//fmt.Println("Got peer event\n")
		fmt.Printf("delete previously peer delete event\n%+v\n%+v", e.Veth, e.Event)
		if e.Event == VPDelete {
			e.fireChangeEvents(VethDelete)
			event.fireChangeEvents(VethDelete)
			n.topology.RemoveFromBuffer(event.GetPeerIndex())
			n.topology.Disconnect(getNSIndex(e.Namespace, e.Index), getNSIndex(event.Namespace, event.Index))
			return
		}
	}
	n.topology.AddToBuffer(event)
}

func (n *Namespace) Connect(ns string) {
	if n != nil {
		if nTemp, ok := n.Connections[ns]; ok {
			if nTemp == ns {
				return
			}
		}
		n.Connections[ns] = ns
		n.fire(NSConnect)
		//peerNs := n.topology.Get(ns)
		//if peerNs != nil {
		//	n.Connections[ns] = peerNs.Name
		//	n.fire(NSConnect)
		//}
	} else {
		fmt.Println("\n\nnamespace", ns, "nil\n\n")
	}
	return
}

func (n *Namespace) Disconnect(ns string) {
	fmt.Println("\n\nDisconnecting", n.Name, ns)
	if _, ok := n.Connections[ns]; ok {
		delete(n.Connections, ns)
		n.fire(NSDisconnect)
	}
	return
}

func (n *Namespace) GetVeth(dev int) *Veth {
	if n != nil {
		if d, ok := n.L2Devices[dev]; ok {
			if v, ok := d.(*Veth); ok {
				return v
			}
			return nil
		}
		return nil
	}
	return nil
}

func (n *Namespace) SetVethPeer(dev int, peerIndex int, peerName, peerNamespace string) {
	if v := n.GetVeth(dev); v != nil {
		v.Pair(peerIndex, peerName, peerNamespace)
	}
	return
}

func (n *Namespace) SetType(s string) {
	if n.Type == s {
		return
	}
	n.Type = s
	n.fire(NSTypeChange)
}

func (n *Namespace) Delete() {
	for _, r := range n.Routes {
		n.DeleteRoute(r)
	}
	n.fire(NSDelete)
}
func (n *Namespace) AddRoute(route netlink.Route) {
	for _, r := range n.Routes {
		if route.Equal(r) {
			return
		}
	}
	n.Routes = append(n.Routes, route)
	n.fire(NSRouteAdd)
}

func (n *Namespace) DeleteRoute(route netlink.Route) {
	for i, r := range n.Routes {
		if route.Equal(r) {
			n.Routes = append(n.Routes[0:i], n.Routes[i+1:]...)
			n.fire(NSRouteDelete)
		}
	}
}
