package devices

import (
	"io"
	"time"
	"github.com/vishvananda/netns"
	"fmt"
)

type Topology struct {
	Namespaces map[string]*Namespace
	buffer     map[string]PeerEvent
}

func NewTopology() *Topology {
	return &Topology{
		make(map[string]*Namespace),
		make(map[string]PeerEvent),
	}
}

func (t *Topology) GetNamespaces() map[string]*Namespace {
	return t.Namespaces
}

var dumper io.Writer

func SetWriter(w io.Writer) {
	dumper = w
	return
}

func (t *Topology) GetDefaultNamespace() *Namespace {
	if n, ok := t.Namespaces["default"]; ok {
		return n
	}
	defaultNS, err := netns.Get()
	if err != nil {
		fmt.Println("ERROR: GETTING CURRENT NS: ", err)
		return nil
	}
	return t.CreateNamespace("default", &defaultNS)
}

func (t *Topology) CreateNamespace(namespace string, targetNs *netns.NsHandle) *Namespace {
	n := NewNamespace(namespace, t, targetNs)
	t.Namespaces[namespace] = &n
	return &n
}

func (t *Topology) Get(namespace string) *Namespace {
	if n, ok := t.Namespaces[namespace]; ok {
		return n
	}
	return nil
}

func (t *Topology) DeleteNamespace(namespace string) {
	if n, ok := t.Namespaces[namespace]; ok {
		<-time.After(100 * time.Millisecond)
		for index, _ := range n.L2Devices {
			n.RemoveDevice(index)
		}
		for index, _ := range n.L3Devices {
			n.RemoveDevice(index)
		}
		n.Delete()
	}
	delete(t.Namespaces, namespace)
}

func (t *Topology) AddToBuffer(event PeerEvent) {
	t.buffer[event.GetIndex()] = event
}

func (t *Topology) GetPeerEvent(event PeerEvent) (PeerEvent, bool) {
	if e, ok := t.buffer[event.GetPeerIndex()]; ok {
		t.RemoveFromBuffer(e.GetIndex())
		return e, ok
	}
	return PeerEvent{}, false
}

func (t *Topology) RemoveFromBuffer(index string) {
	delete(t.buffer, index)
}

func (t *Topology) Connect(ns1, ns2 string) {
	t.Get(ns1).Connect(ns2)
	t.Get(ns2).Connect(ns1)
	fmt.Println("NS:", ns1[:7], "and NS:", ns2[:7], "connected")
}

func (t *Topology) Disconnect(ns1, ns2 string) {
	n1 := t.Get(ns1)
	n2 := t.Get(ns2)
	if n1 != nil {
		n1.Disconnect(ns2)
	}
	if n2 != nil {
		n2.Disconnect(ns1)
	}
	fmt.Println("NS:", ns1[:7], "and NS:", ns2[:7], "disconnected")
}
