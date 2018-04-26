package devices

import (
	"errors"
	"fmt"
	"net"

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
)

var NSEventStrings = []string{
	"NSCreate",
	"NSDelete",
	"NSTypeChange",
	"NSConnect",
	"NSDisconnect",
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
	n := Namespace{
		Name:           name,
		L2Devices:      make(map[int]LinkUpdateReceiver),
		L3Devices:      make(map[int]LinkAddrUpdateReceiver),
		Connections:    make(map[string]string),
		nsHandle:       targetNs,
		topology:       t,
		onchange:       make(map[NSEvent][]func(*Namespace, NSEvent)),
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
		*d.l3EventChannel().addAddrChannel <- addr
		n.SetType("network")
	}
}

func (n *Namespace) RemoveL3Addr(index int, addr *net.IPNet) {
	if d, ok := n.L3Devices[index]; ok {
		*d.l3EventChannel().removeAddrChannel <- addr
	}
}

func (n *Namespace) RemoveDevice(index int) {
	if dev, ok := n.L3Devices[index]; ok {
		*dev.l3EventChannel().doneChannel <- true
		<-*dev.l3EventChannel().doneChannel
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
		*(d.l2EventChannel().flagsChannel) <- e

	}
	//TODO
	//if d, ok := topology.L3Devices[index]; ok {
	//}
}

func (n *Namespace) SetMaster(devIndex int, masterIndex int) {
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

func (n *Namespace) RemoveMaster(devIndex int, masterIndex int) {
	if d, ok := n.L2Devices[devIndex]; ok {
		if m, ok := n.L2Devices[masterIndex]; ok {
			e := newL2DeviceMasterEvent(devIndex, 0)
			*(d.l2EventChannel().masterChannel) <- e
			*(m.l2EventChannel()).masterChannel <- e
		}
	}
}

func (n *Namespace) Dump(index int) {
	if d, ok := n.L2Devices[index]; ok {
		*d.l2EventChannel().dump <- true
		<-*d.l2EventChannel().dump
		if l3d, ok := n.L3Devices[index]; ok {
			*l3d.l3EventChannel().dumpChannel <- true
			<-*l3d.l3EventChannel().dumpChannel
		}
	} else {
		fmt.Println("No device with the index")
	}
}

func (n *Namespace) DumpAll() {
	for _, value := range n.L3Devices {
		*value.l3EventChannel().dumpChannel <- true
		_ = <-*value.l3EventChannel().dumpChannel
	}
	for index, _ := range n.L2Devices {
		if dev, ok := n.L2Devices[index]; !ok {
			*dev.l2EventChannel().dump <- true
			<-*dev.l2EventChannel().dump
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
		*d.l2EventChannel().nameChannel <- newName
	}
}

func (n *Namespace) SendVPCreateEvent(event PeerEvent) {
	//fmt.Printf("CREATE %+v %+v\n", *event.Veth, event.Event)
	if e, ok := n.topology.GetEvent(event); ok {
		if e.Event == VPDelete {
			//fmt.Println("create Got own event\n")
			n.topology.RemoveFromBuffer(event.GetIndex())
			n.topology.Connect(e.PeerNamespace, event.Namespace)
			event.Pair(e.PeerIndex, e.PeerName, e.PeerNamespace)
			n.topology.Get(e.PeerNamespace).SetVethPeer(e.PeerIndex, event.Veth)
			return
		}
	} else if e, ok := n.topology.GetPeerEvent(event); ok {
		//fmt.Println("Create Got peer event\n")
		if e.Event == VPCreate {
			n.topology.RemoveFromBuffer(event.GetPeerIndex())
			n.topology.Connect(event.Namespace, e.Namespace)
			event.Pair(e.Index, e.Name, e.Namespace)
			e.Pair(event.Index, event.Name, event.Namespace)
			return
		}
	}
	n.topology.AddToBuffer(event)
}

func (n *Namespace) SendVPDeleteEvent(event PeerEvent) {
	//if e, ok := n.topology.buffer[event.GetIndex()]; ok {
	//	//n.topology.Connect()
	//}
	//fmt.Printf("Delete %+v %+v\n", *event.Veth, event.Event)
	if e, ok := n.topology.GetEvent(event); ok {
		//fmt.Println("Got own event\n")
		if e.Event == VPCreate {
			n.topology.RemoveFromBuffer(event.GetIndex())
			e.Pair(event.Index, event.Name, event.Namespace)
			n.topology.Get(e.PeerNamespace).SetVethPeer(e.PeerIndex, e.Veth)
			n.topology.Connect(e.Namespace, event.PeerNamespace)
			return
		}
	} else if e, ok := n.topology.GetPeerEvent(event); ok {
		//fmt.Println("Got peer event\n")
		if e.Event == VPDelete {
			e.fireChangeEvents(VethDelete)
			event.fireChangeEvents(VethDelete)
			n.topology.RemoveFromBuffer(event.GetPeerIndex())
			n.topology.Disconnect(e.Namespace, event.Namespace)
			return
		}
	}
	n.topology.AddToBuffer(event)
}

func (n *Namespace) Connect(ns string) {
	if nTemp, ok := n.Connections[ns]; ok {
		if nTemp == ns {
			return
		}
	}
	n.Connections[ns] = (*n.topology.Get(ns)).Name
	n.fire(NSConnect)
	return
}

func (n *Namespace) Disconnect(ns string) {
	if nTemp, ok := n.Connections[ns]; ok {
		if nTemp == ns {
			delete(n.Connections, ns)
		}
	}
	n.fire(NSDisconnect)
	return
}

func (n *Namespace) GetVeth(dev int) *Veth {
	if d, ok := n.L2Devices[dev]; ok {
		if v, ok := d.(*Veth); ok {
			return v
		}
		return nil
	}
	return nil
}

func (n *Namespace) SetVethPeer(dev int, peer *Veth) {
	if v := n.GetVeth(dev); v != nil {
		v.Pair(peer.Index, peer.Name, peer.Namespace)
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
	n.fire(NSDelete)
}
