package devices

import (
	"testing"
	"github.com/vishvananda/netlink"
)

func TestL2Bridge_RemovePort(t *testing.T) {
	update := netlink.LinkUpdate{
		nil,
		nil,
		&netlink.Dummy{netlink.LinkAttrs{Name: "foo"}},
	}
	l2Bridge := NewL2Bridge(update)
	//should work without any changes
	l2Bridge.RemovePort(1)
	if len(l2Bridge.Ports) != 0 {
		t.Fail()
	}
	l2Bridge.AddPort(1)
	l2Bridge.RemovePort(1)
	if len(l2Bridge.Ports) != 0 {
		t.Fail()
	}
	l2Bridge.AddPort(1)
	l2Bridge.AddPort(2)
	l2Bridge.AddPort(3)

	l2Bridge.RemovePort(2)
	if len(l2Bridge.Ports) != 2 {
		t.Fail()
	}
	if l2Bridge.Ports[0] != 1 || l2Bridge.Ports[1] != 2 {
		t.Fail()
	}
}
