package devices

import "strconv"

type VethPairEvent int

const (
	VPCreate = iota
	VPDelete
)

type PeerEvent struct {
	*Veth
	Event		  VethPairEvent
}

func NewPeerEvent (v *Veth, event VethPairEvent) PeerEvent {
	return PeerEvent{
		v,
		event,
	}
}

func (p PeerEvent) GetIndex() string {
	return strconv.Itoa(p.Index) + ":" + strconv.Itoa(p.PeerIndex)
}

func (p PeerEvent) GetPeerIndex() string {
	return strconv.Itoa(p.PeerIndex) + ":" + strconv.Itoa(p.Index)
}
