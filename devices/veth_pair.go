package devices

import "strconv"

type VethPairEvent int

const (
	VPCreate = iota
	VPDelete
)

type PeerEvent struct {
	Index         int
	PeerIndex     int
	PeerNamespace string
	Namespace     string
	Event		  VethPairEvent
}

func NewPeerEvent (index, peerIndex int, namespace, peerNamespace string, event VethPairEvent) PeerEvent {
	return PeerEvent{
		Index:index,
		PeerIndex:peerIndex,
		Namespace:namespace,
		PeerNamespace:peerNamespace,
		Event:event,
	}
}

func (p PeerEvent) GetIndex() string {
	return strconv.Itoa(p.Index) + ":" + strconv.Itoa(p.PeerIndex)
}

func (p PeerEvent) GetPeerIndex() string {
	return strconv.Itoa(p.PeerIndex) + ":" + strconv.Itoa(p.Index)
}
