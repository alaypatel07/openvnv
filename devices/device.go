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
	l3EventChannel() l3Channel
}

type LinkAddrUpdateReceiver interface {
	LinkUpdateReceiver
	AddrUpdateReceiver
}

type Transformer interface {
	Transform(*netlink.AddrUpdate) *L3Device
}
