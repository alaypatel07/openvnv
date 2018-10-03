package devices

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	VethPair = iota
	VethUnknown
	VethDelete
)

var VethEventStrings = []string{
	"VethPair",
	"VethUnknown",
	"VethDelete",
}

type VethEvent int

func (e VethEvent) String() string {
	for i, str := range VethEventStrings {
		if i == int(e) {
			return str
		}
	}
	return ""
}

type Veth struct {
	*L2Device
	PeerName      string
	PeerIndex     int
	PeerNamespace string
	onChange      map[VethEvent][]func(*Veth, VethEvent)
}

var defaultVethSubscriber []func(*Veth, VethEvent)

func SubscribeAllVethEvents(callback func(*Veth, VethEvent)) {
	defaultVethSubscriber = append(defaultVethSubscriber, callback)
}

func NewVeth(update netlink.Link, t *Topology, namespace string, consoleDisplay bool) (*Veth, error) {
	l2dev := NewL2Device(update, t, namespace, consoleDisplay)
	v := &Veth{
		L2Device:  l2dev,
		PeerIndex: -1,
		onChange:  make(map[VethEvent][]func(veth *Veth, event VethEvent)),
	}
	for index, _ := range VethEventStrings {
		for _, defaultCallback := range defaultVethSubscriber {
			if err := v.OnChange(VethEvent(index), defaultCallback); err != nil {
				fmt.Println("ERROR: ASSIGNING ONCHANGE", err)
			}
		}
	}
	func() {
		//<- time.After(100 * time.Millisecond)
		if peer, ok := v.isPeerVisible(); ok {
			v.PeerNamespace = v.Namespace
			v.PeerName = peer.Name
			v.PeerIndex = peer.Index
			v.raiseCreateEvent()
			//v.Pair(peer.Index, peer.Name, v.Namespace)
		} else {
			//fmt.Println(v.Index, v.PeerIndex, v.Namespace, "VETH CREATE")
			v.raiseCreateEvent()
		}
	}()
	return v, nil
}

func (v *Veth) Pair(peerIndex int, peerName, peerNamespace string) {
	//fmt.Println("Pairing", v.Name, v.Namespace[:7], "with", peer.Name, peer.Namespace[:7])
	v.PeerNamespace = peerNamespace
	v.PeerName = peerName
	v.PeerIndex = peerIndex
	v.fireChangeEvents(VethPair)
}

func (v *Veth) Unknown() {
	v.fireChangeEvents(VethUnknown)
}

func (v *Veth) raiseCreateEvent() {
	e := NewPeerEvent(v, VPCreate)
	v.topology.Get(v.Namespace).SendVPCreateEvent(e)
}

func (v *Veth) raiseDeleteEvent() {
	e := NewPeerEvent(v, VPDelete)
	//fmt.Printf("Raise Delete Event %+v\n", e.Veth)
	v.topology.Get(v.Namespace).SendVPDeleteEvent(e)
}

func (v *Veth) isPeerVisible() (*netlink.Veth, bool) {
	//<- time.After(50 * time.Millisecond)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	err := netns.Set(*v.topology.Get(v.Namespace).nsHandle)
	if err != nil {
		fmt.Println("ERROR: SETTING NS TO ", v.Namespace, err)
		return nil, false
	}
	t, err := netlink.LinkByIndex(v.Index)
	if err != nil {
		fmt.Println("ERROR: GETTING LINK", v.Namespace, err)
		return nil, false
	}
	v.Name = t.Attrs().Name
	d, err := netlink.VethPeerIndex(t.(*netlink.Veth))
	if err != nil {
		//fmt.Println("ERROR: GETTING PEER INDEX in NS", v.Namespace, err)
		<-time.After(500 * time.Millisecond)
		d, err = netlink.VethPeerIndex(t.(*netlink.Veth))
		if err != nil {
			fmt.Println("ERROR: GETTING PEER INDEX in NS even after 500 milliseconds", v.Namespace, err, t.Attrs().ParentIndex)
			return nil, false
		}
	}
	v.SetPeerIndex(d)
	p, err := netlink.LinkByIndex(d)
	if err != nil {
		return nil, false
	}
	return p.(*netlink.Veth), true
}

func (v *Veth) ReceiveLinkUpdate() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	netns.Set(*v.topology.Get(v.Namespace).nsHandle)
	for {
		select {
		case f := <-*(v.flagsChannel):
			v.SetFlags(f.flags, f.operState)
		case m := <-*(v.setMasterChannel):
			if m.masterIndex != 0 {
				v.SetMaster(m.masterIndex)
			} else {
				v.UnsetMaster()
			}
		case d := <-*(v.dumpChannel):
			if d {
				dumper.Encode(v)
				*(v.dumpChannel) <- true
			}
		case d := <-*(v.deleteChannel):
			if d {
				return
			}
		case n := <-*(v.nameChannel):
			v.SetName(n)

		}
	}
}

func (v *Veth) DeleteDevice() {
	//fmt.Println("Delete VETH")
	v.L2Device.DeleteDevice()
	//fmt.Println("Index:", v.Index, " Peer Index:", v.PeerIndex, "Namespace:", v.Namespace, "Peer Namespace:, ", v.PeerNamespace, "VETH DELETE")
	v.raiseDeleteEvent()
}

func (v *Veth) SetName(s string) {
	if v.Name == s {
		return
	}
	v.Name = s
}
func (v *Veth) SetPeerIndex(i int) {
	v.PeerIndex = i
}
func (v *Veth) OnChange(event VethEvent, callback func(*Veth, VethEvent)) error {
	if int(event) >= len(VethEventStrings) || int(event) < 0 {
		return errors.New("Veth OnChange: VethEvent unrecognized")
	}
	v.onChange[event] = append(v.onChange[event], callback)
	return nil
}
func (v *Veth) fireChangeEvents(event VethEvent) {
	for _, f := range v.onChange[event] {
		f(v, event)
	}
}
