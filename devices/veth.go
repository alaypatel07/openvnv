package devices

import (
	"github.com/vishvananda/netlink"
	"os"
	"encoding/json"
	"fmt"
	"runtime"
	"github.com/vishvananda/netns"
)

type Veth struct {
	*L2Device
	PeerName      string
	PeerIndex     int
	peerNamespace string
}

func NewVeth(update netlink.Link, t *Topology,namespace string, consoleDisplay bool) (*Veth, error) {
	l2dev := NewL2Device(update, t, namespace, consoleDisplay)
	v := &Veth{
		l2dev,
		"",
		-1,
		"",
	}
	func() {
		if peer, ok := v.isPeerVisible(); ok {
			v.PeerName = peer.Name
			v.PeerIndex = peer.Index
			v.peerNamespace = v.Namespace
		} else {
			//fmt.Println(v.Index, v.PeerIndex, v.Namespace, "VETH CREATE")
			v.raiseCreateEvent()
		}
	}()
	return v, nil
}

func (v *Veth) raiseCreateEvent() {
	e := NewPeerEvent(v.Index, v.PeerIndex, v.Namespace, "", VPCreate)
	v.topology.Get(v.Namespace).SendVPCreateEvent(e)
}

func (v *Veth) raiseDeleteEvent() {
	e := NewPeerEvent(v.Index, v.PeerIndex, v.Namespace, v.peerNamespace, VPDelete)
	v.topology.Get(v.Namespace).SendVPDeleteEvent(e)
}

func (v *Veth) isPeerVisible() (*netlink.Veth, bool) {
	//<- time.After(50 * time.Millisecond)
	err :=  netns.Set(*v.topology.Get(v.Namespace).nsHandle)
	if err != nil {
		fmt.Println("ERROR: SETTING NS TO ", v.Namespace, err)
		return nil, false
	}
	t, err := netlink.LinkByIndex(v.Index)
	if err != nil {
		fmt.Println("ERROR: GETTING LINK", v.Namespace, err)
		return nil, false
	}
	d, err := netlink.VethPeerIndex(t.(*netlink.Veth))
	if err != nil {
		fmt.Println("ERROR: GETTING PEER INDEX in NS", v.Namespace, err)
		return nil, false
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
				if dumper != os.Stdout {
					json.NewEncoder(dumper).Encode(v)
				}
				json.NewEncoder(os.Stdout).Encode(v)
				*(v.dumpChannel) <- true
			}
		case d := <-*(v.deleteChannel):
			if d {
				return
			}
		case n := <- *(v.nameChannel):
			v.SetName(n)

		}
	}
}

func (v *Veth) DeleteDevice() {
	*v.deleteChannel <- true
	v.fireChangeEvents(L2DeviceDelete)
	//fmt.Println("Index:", v.Index, " Peer Index:", v.PeerIndex, "Namespace:", v.Namespace, "Peer Namespace:, ", v.peerNamespace, "VETH DELETE")
	v.raiseDeleteEvent()
}

func (v *Veth) SetName(s string) {
	if v.Name == s {
		return
	}
	v.Name = s
}
func (veth *Veth) SetPeerIndex(i int) {
	veth.PeerIndex = i
}

