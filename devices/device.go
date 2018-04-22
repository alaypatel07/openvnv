package devices

import (
	"github.com/vishvananda/netlink"
)

type LinkUpdateReceiver interface {
	ReceiveLinkUpdate()
	l2EventChannel() l2Channel
}

type AddrUpdateReceiver interface {
	ReceiveAddrUpdate()
	L3EventChannel() chan interface{}
}

type LinkAddrUpdateReceiver interface {
	LinkUpdateReceiver
	AddrUpdateReceiver
}

type Transformer interface {
	Transform(*netlink.AddrUpdate) *L3Device
}
