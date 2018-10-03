package devices

import (
	"errors"
	"fmt"
	"net"
)

type L3DeviceEvent int

func (e L3DeviceEvent) String() string {
	for i, str := range L3DeviceEventStrings {
		if e == L3DeviceEvent(i) {
			return str
		}
	}
	return ""
}

const (
	L3DeviceCreate = iota
	L3DeviceAddAddress
	L3DeviceRemoveAddress
	L3DeviceDelete
)

var L3DeviceEventStrings = []string{
	"L3DeviceCreate",
	"L3DeviceAddAddress",
	"L3DeviceRemoveAddress",
	"L3DeviceDelete",
}

type L3Channel struct {
	addAddrChannel    *chan *net.IPNet
	removeAddrChannel *chan *net.IPNet
	dumpChannel       *chan bool
	doneChannel       *chan bool
}

func newL3Channel() L3Channel {
	a := make(chan *net.IPNet)
	r := make(chan *net.IPNet)
	d := make(chan bool)
	dc := make(chan bool)
	return L3Channel{addAddrChannel: &a, removeAddrChannel: &r, dumpChannel: &d, doneChannel: &dc}
}

type L3Device struct {
	Index     int
	Namespace string
	LinkUpdateReceiver
	IP          []string
	ip          []*net.IPNet
	onChange    map[L3DeviceEvent][]func(device *L3Device, event L3DeviceEvent)
	addrChannel L3Channel
}

var defaultL3DeviceSubscriber []func(device *L3Device, event L3DeviceEvent)

func SubscribeAllL3DeviceEvents(callback func(device *L3Device, event L3DeviceEvent)) {
	defaultL3DeviceSubscriber = append(defaultL3DeviceSubscriber, callback)
}

func (dev *L3Device) ReceiveAddrUpdate() {
	for {
		select {
		case a := <-*(dev.addrChannel.addAddrChannel):
			dev.AddAddr(a)
		case a := <-*dev.addrChannel.removeAddrChannel:
			dev.RemoveAddr(a)
		case d := <-*dev.L3EventChannel().dumpChannel:
			if d {
				dumper.Encode(dev)
				*dev.L3EventChannel().dumpChannel <- true
			}
		case d := <-*dev.L3EventChannel().doneChannel:
			if d {
				for _, addr := range dev.ip {
					dev.RemoveAddr(addr)
				}
				dev.fireChangeEvents(L3DeviceDelete)
				*dev.L3EventChannel().doneChannel <- true
				*dev.L3EventChannel().doneChannel <- true
				return
			}
		}
	}
}

func (dev *L3Device) L3EventChannel() L3Channel {
	return dev.addrChannel
}

func NewL3Device(index int, namespace string, l2dev LinkUpdateReceiver, addrs []*net.IPNet, consoleDisplay bool) *L3Device {
	d := &L3Device{
		Index:              index,
		Namespace:          namespace,
		LinkUpdateReceiver: l2dev,
		addrChannel:        newL3Channel(),
		onChange:           make(map[L3DeviceEvent][]func(device *L3Device, event L3DeviceEvent)),
	}
	for index, _ := range L3DeviceEventStrings {
		for _, defaultCallback := range defaultL3DeviceSubscriber {
			if err := d.OnChange(L3DeviceEvent(index), defaultCallback); err != nil {
				fmt.Println("ERROR: ASSIGNING ONCHANGE", err)
			}
		}
	}
	d.fireChangeEvents(L3DeviceCreate)
	for _, addr := range addrs {
		d.AddAddr(addr)
	}
	go d.ReceiveAddrUpdate()
	return d
}

func (dev *L3Device) AddAddr(addr *net.IPNet) {
	if dev.IP == nil {
		dev.ip = make([]*net.IPNet, 1)
		dev.IP = make([]string, 1)
		dev.IP[0] = addr.String()
		dev.ip[0] = addr
		dev.fireChangeEvents(L3DeviceAddAddress)
		return
	}
	dev.IP = append(dev.IP, addr.String())
	dev.ip = append(dev.ip, addr)
	dev.fireChangeEvents(L3DeviceAddAddress)
}

func (dev *L3Device) RemoveAddr(addr *net.IPNet) {
	for index, ip := range dev.ip {
		sizea, _ := ip.Mask.Size()
		sizeb, _ := addr.Mask.Size()
		// ignore label for comparison
		if ip.IP.Equal(addr.IP) && sizea == sizeb {
			dev.IP = append(dev.IP[0:index], dev.IP[index+1:]...)
			dev.ip = append(dev.ip[0:index], dev.ip[index+1:]...)
			dev.fireChangeEvents(L3DeviceRemoveAddress)
			return
		}
	}
}

func (dev *L3Device) OnChange(event L3DeviceEvent, callback func(device *L3Device, event L3DeviceEvent)) error {
	if int(event) >= len(L3DeviceEventStrings) || int(event) < 0 {
		return errors.New("L3Device OnChange: L3DeviceEvent unrecognized")
	}
	dev.onChange[event] = append(dev.onChange[event], callback)
	return nil
}

func (dev *L3Device) fireChangeEvents(change L3DeviceEvent) {
	for _, f := range dev.onChange[change] {
		f(dev, change)
	}
}
