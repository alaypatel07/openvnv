package devices

import "github.com/vishvananda/netlink"

type Interface struct {
	*L2Device
}

func NewInterface(update netlink.LinkUpdate, consoleDisplay bool) Interface {
	return Interface{
		NewL2Device(&update, consoleDisplay),
	}
}