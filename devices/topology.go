package devices

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/vishvananda/netns"
)

type Topology struct {
	Namespaces map[string]*Namespace
	buffer     map[string]PeerEvent
	sync.Mutex
}

func NewTopology() *Topology {
	t := Topology{
		Namespaces: make(map[string]*Namespace),
		buffer:     make(map[string]PeerEvent),
	}
	return &t
}

func (t *Topology) GetNamespaces() map[string]*Namespace {
	return t.Namespaces
}

var dumper *json.Encoder

func SetWriter(w io.Writer) {
	dumper = json.NewEncoder(w)
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
	t.Lock()
	t.buffer[event.GetIndex()] = event
	event.fireChangeEvents(VethUnknown)
	t.Unlock()
}

func (t *Topology) GetPeerEvent(event PeerEvent) (PeerEvent, bool) {
	t.Lock()
	if e, ok := t.buffer[event.GetPeerIndex()]; ok {
		t.Unlock()
		return e, ok
	}
	t.Unlock()
	return PeerEvent{}, false
}

func (t *Topology) GetEvent(event PeerEvent) (PeerEvent, bool) {
	t.Lock()
	if e, ok := t.buffer[event.GetIndex()]; ok {
		t.Unlock()
		return e, ok
	}
	t.Unlock()
	return PeerEvent{}, false
}

func (t *Topology) RemoveFromBuffer(index string) {
	t.Lock()
	delete(t.buffer, index)
	t.Unlock()
}

func (t *Topology) Connect(ns1, ns2 string) {
	t.Get(ns1).Connect(ns2)
	t.Get(ns2).Connect(ns1)
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
}
