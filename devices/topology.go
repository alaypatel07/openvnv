package devices

import (
	"io"
	"time"
)

type Topology map[string]*Namespace

func NewTopology() Topology {
	return Topology(make(map[string]*Namespace))
}

var dumper io.Writer

func SetWriter(w io.Writer) {
	dumper = w
	return
}

func (t Topology) GetDefaultNamespace() *Namespace {
	if n, ok := t["default"]; ok {
		return n
	}
	return t.CreateNamespace("default")
}

func (t Topology) CreateNamespace(namespace string) *Namespace {
	n := NewNamespace(namespace)
	t[namespace] = &n
	return &n
}

func (t Topology) Get(namespace string) *Namespace {
	if n, ok := t[namespace]; ok {
		return n
	}
	return nil
}

func (t Topology) DeleteNamespace(namespace string) {
	if n, ok := t[namespace]; ok {
		<- time.After(100 * time.Millisecond)
		for index, _ := range n.L2Devices {
			n.RemoveDevice(index)
		}
		for index, _ := range n.L3Devices {
			n.RemoveDevice(index)
		}
		n.Delete()
	}
	delete(t, namespace)
}
