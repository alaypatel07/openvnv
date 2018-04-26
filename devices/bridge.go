package devices

import (
	"net"

	"github.com/vishvananda/netlink"
)

type Bridge interface {
	AddPort(int)
	RemovePort(int)
}

type L2BridgeEvent int

const bridgeIota int = 8

const (
	L2BridgeCreate L2BridgeEvent = iota + L2BridgeEvent(bridgeIota)
	L2BridgeDelete
	L2BridgeAddPort
	L2BridgeRemovePort
)

var L2BridgeEventStrings = []string{
	"L2BridgeCreate",
	"L2BridgeDelete",
	"L2BridgeAddPort",
	"L2BridgeDeletePort",
}

func (e L2BridgeEvent) String() string {
	for i, str := range L2BridgeEventStrings {
		if i == int(e-L2BridgeEvent(bridgeIota)) {
			return str
		}
	}
	return ""
}

type L2Bridge struct {
	*L2Device
	Ports         map[int]int
	onchange      map[L2BridgeEvent][]func(dev L2Bridge, event L2BridgeEvent)
	masterChannel *chan l2DeviceMasterEvent
	BridgeEvent   string `json:"bridge_event"`
}

func (dev *L2Bridge) AddPort(devIndex int) {
	dev.Ports[len(dev.Ports)] = devIndex
	dev.fireChangeEvents(L2BridgeAddPort)
}

func (dev *L2Bridge) RemovePort(devIndex int) {
	for index, value := range dev.Ports {
		if value == devIndex {
			delete(dev.Ports, index)
			dev.fireChangeEvents(L2BridgeRemovePort)
		}
	}
}

func NewL2Bridge(update netlink.Link, t *Topology, namespace string, consoleDisplay bool) *L2Bridge {
	defaultFunction := func(dev L2Bridge, change L2BridgeEvent) {
		dev.BridgeEvent = change.String()
		dumper.Encode(dev)
	}
	onChange := make(map[L2BridgeEvent][]func(dev L2Bridge, change L2BridgeEvent))
	if consoleDisplay {
		for i, _ := range L2BridgeEventStrings {
			onChange[L2BridgeEvent(i)] = append(onChange[L2BridgeEvent(i)], defaultFunction)
		}
	}
	l2br := &L2Bridge{
		L2Device: NewL2Device(update, t, namespace, consoleDisplay),
		Ports:    make(map[int]int),
		onchange: onChange,
	}
	l2br.CreateDevice()
	return l2br
}

func (dev *L2Bridge) fireChangeEvents(change L2BridgeEvent) {
	for _, f := range dev.onchange[change-L2BridgeEvent(bridgeIota)] {
		f(*dev, change)
	}
}

func (dev *L2Bridge) CreateDevice() {
	dev.fireChangeEvents(L2BridgeCreate)
}

func (dev *L2Bridge) DeleteDevice() {
	dev.L2Device.DeleteDevice()
	dev.fireChangeEvents(L2BridgeDelete)
}

func (dev *L2Bridge) ReceiveLinkUpdate() {
	for {
		select {
		case m := <-*(dev.flagsChannel):
			dev.SetFlags(m.flags, m.operState)
		case m := <-*(dev.setMasterChannel):
			if m.masterIndex != 0 {
				dev.AddPort(m.devIndex)
			} else {
				dev.RemovePort(m.devIndex)
			}
		case d := <-*(dev.dumpChannel):
			if d {
				dumper.Encode(dev)
				*(dev.dumpChannel) <- true
			}
		case d := <-*(dev.deleteChannel):
			if d {
				return
			}
		case n := <-*(dev.nameChannel):
			dev.SetName(n)
		}
	}
}

type L3Bridge struct {
	L2Bridge
	Addr net.IPNet
}

func NewL3Bridge(l2 L2Bridge, update netlink.AddrUpdate) *L3Bridge {
	return &L3Bridge{
		l2,
		update.LinkAddress,
	}
}
