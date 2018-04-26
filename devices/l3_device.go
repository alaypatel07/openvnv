package devices

import "net"

type L3Device struct {
	*L2Device
	Ip net.IPNet
}
