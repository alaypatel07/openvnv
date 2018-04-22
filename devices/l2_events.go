package devices

import (
	"github.com/vishvananda/netlink"
	"net"
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

type l2Channel struct {
	master        int
	masterChannel *chan l2DeviceMasterEvent
	flagsChannel  *chan l2DeviceFlagsEvent
	dump          *chan bool
}

func newL2Channel(masterIndex int, m *chan l2DeviceMasterEvent, f *chan l2DeviceFlagsEvent, d *chan bool) l2Channel {
	return l2Channel{masterIndex, m, f, d}
}
