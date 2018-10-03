package devices

import (
	"net"

	"github.com/vishvananda/netlink"
)

type l2DeviceFlagsEvent struct {
	flags     net.Flags
	operState netlink.LinkOperState
}

type l2DeviceMasterEvent struct {
	devIndex    int
	masterIndex int
}

func newL2DeviceFlagsEvent(f net.Flags, o netlink.LinkOperState) l2DeviceFlagsEvent {
	return l2DeviceFlagsEvent{f, o}
}

func newL2DeviceMasterEvent(devIndex int, masterIndex int) l2DeviceMasterEvent {
	return l2DeviceMasterEvent{devIndex, masterIndex}
}

type L2channel struct {
	Master        int
	masterChannel *chan l2DeviceMasterEvent
	flagsChannel  *chan l2DeviceFlagsEvent
	nameChannel   *chan string
	dump          *chan bool
}

func newL2Channel(masterIndex int, m *chan l2DeviceMasterEvent, f *chan l2DeviceFlagsEvent, n *chan string,
	d *chan bool) L2channel {
	return L2channel{masterIndex, m, f, n, d}
}
