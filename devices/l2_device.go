package devices

import (
	"net"

	"github.com/vishvananda/netlink"
)

type L2Event int

const (
	L2DeviceCreate L2Event = iota
	L2DeviceDelete
	L2DeviceUp
	L2DeviceDown
	L2DeviceLowerLayerDown
	L2DeviceSetMaster
	L2DeviceUnsetMaster
	L2DeviceTransform
)

var L2EventStrings = []string{
	"L2 DeviceCreated",
	"L2DeviceDeleted",
	"L2DeviceUp",
	"L2DeviceDown",
	"L2DeviceLowerLayerDown",
	"L2DeviceSetMaster",
	"L2DeviceUnsetMaster",
	"L2DeviceTransform",
}

func (e L2Event) String() string {
	for i, str := range L2EventStrings {
		if i == int(e) {
			return str
		}
	}
	return ""
}

type L2Status int

const (
	L2Down L2Status = iota
	L2Up
	L2LowerLayerDown
)

var L2StatusStrings = []string{
	"DOWN",
	"UP",
	"LOWERLAYERDOWN",
}

func (s L2Status) String() string {
	for i, str := range L2StatusStrings {
		if i == int(i) {
			return str
		}
	}
	return ""
}

type L2Device struct {
	topology         *Topology
	Name             string   `json:"name"`
	Index            int      `json:"index"`
	Status           L2Status `json:"status"`
	Master           int      `json:"Master"`
	Namespace        string   `json:"namespace"`
	flags            net.Flags
	operState        netlink.LinkOperState
	onchange         map[L2Event][]func(dev L2Device, change L2Event)
	flagsChannel     *chan l2DeviceFlagsEvent
	setMasterChannel *chan l2DeviceMasterEvent
	deleteChannel    *chan bool
	nameChannel      *chan string
	dumpChannel      *chan bool
	Event            string `json:"event"`
}

func NewL2Device(update netlink.Link, t *Topology, namespace string, consoleDisplay bool) *L2Device {
	defaultFunction := func(dev L2Device, change L2Event) {
		dev.Event = change.String()
		t := make(map[string]interface{})
		t["name"] = dev.Index
		t["indexName"] = "device1"
		t["ns"] = dev.Namespace
		t["connections"] = dev.Master
		switch change {
		case L2DeviceCreate:
			t["event"] = "create"
		case L2DeviceDelete:
			t["event"] = "delete"
		default:
			t["event"] = "update"
		}
		dumper.Encode(t)
	}
	onChange := make(map[L2Event][]func(dev L2Device, change L2Event))
	if consoleDisplay {
		for i, _ := range L2EventStrings {
			onChange[L2Event(i)] = append(onChange[L2Event(i)], defaultFunction)
		}
	}

	l := make(chan l2DeviceFlagsEvent)
	m := make(chan l2DeviceMasterEvent)
	dumpChannel := make(chan bool)
	deleteChannel := make(chan bool)
	nameChannel := make(chan string)
	l2dev := L2Device{
		topology:         t,
		Name:             update.Attrs().Name,
		Index:            update.Attrs().Index,
		Master:           0,
		Namespace:        namespace,
		flags:            net.Flags(0),
		operState:        netlink.OperNotPresent,
		onchange:         onChange,
		flagsChannel:     &l,
		setMasterChannel: &m,
		deleteChannel:    &deleteChannel,
		nameChannel:      &nameChannel,
		dumpChannel:      &dumpChannel,
	}
	l2dev.CreateDevice()
	return &l2dev
}

func (dev *L2Device) L2EventChannel() L2channel {
	return newL2Channel(dev.Master, dev.setMasterChannel, dev.flagsChannel, dev.nameChannel, dev.dumpChannel)
}

func (dev *L2Device) fireChangeEvents(change L2Event) {
	for _, f := range dev.onchange[change] {
		f(*dev, change)
	}
}

func (dev *L2Device) CreateDevice() {
	dev.fireChangeEvents(L2DeviceCreate)
}

func (dev *L2Device) DeleteDevice() {
	*dev.deleteChannel <- true
	dev.fireChangeEvents(L2DeviceDelete)
}

func (dev *L2Device) SetMaster(masterIndex int) {
	if masterIndex == 0 {
		return
	}
	dev.Master = masterIndex
	dev.fireChangeEvents(L2DeviceSetMaster)
}

func (dev *L2Device) UnsetMaster() {
	dev.Master = 0
	dev.fireChangeEvents(L2DeviceUnsetMaster)
}

func (dev *L2Device) Up() {
	dev.Status = L2Up
	dev.fireChangeEvents(L2DeviceUp)
}

func (dev *L2Device) Down() {
	dev.Status = L2Down
	dev.fireChangeEvents(L2DeviceDown)
}

func (dev *L2Device) UpLowerLayerDown() {
	dev.Status = L2LowerLayerDown
	dev.fireChangeEvents(L2DeviceLowerLayerDown)
}

func (dev *L2Device) SetFlags(flags net.Flags, operState netlink.LinkOperState) {
	if dev.flags == flags && dev.operState == operState {
		return
	}
	dev.flags = flags
	dev.operState = operState
	if flags&1 == net.FlagUp {
		if operState == netlink.OperUp || operState == netlink.OperUnknown {
			dev.Up()
		} else if operState == netlink.OperLowerLayerDown {
			dev.UpLowerLayerDown()
		} else if operState == netlink.OperDown {
			dev.Down()
		}
	} else {
		dev.Down()
	}
}

func (dev *L2Device) ReceiveLinkUpdate() {

	for {
		select {
		case f := <-*(dev.flagsChannel):
			dev.SetFlags(f.flags, f.operState)
		case m := <-*(dev.setMasterChannel):
			if m.masterIndex != 0 {
				dev.SetMaster(m.masterIndex)
			} else {
				dev.UnsetMaster()
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

func (device *L2Device) SetName(s string) {
	if device.Name == s {
		return
	}
	device.Name = s
}
