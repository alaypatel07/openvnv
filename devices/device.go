package devices

import (
	"github.com/vishvananda/netlink"
)

type LinkUpdateReceiver interface {
	ReceiveLinkUpdate()
	L2EventChannel() L2channel
}

type AddrUpdateReceiver interface {
	ReceiveAddrUpdate()
	L3EventChannel() L3Channel
}

type LinkAddrUpdateReceiver interface {
	LinkUpdateReceiver
	AddrUpdateReceiver
}

type Transformer interface {
	Transform(*netlink.AddrUpdate) *L3Device
}
