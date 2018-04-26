package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/alaypatel07/openvnv/devices"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

var namespace *devices.Namespace
var topology = devices.NewTopology()
var consoleDisplay *bool

var dumpIP *string
var encoder *json.Encoder

func getEncoder() *json.Encoder {
	if encoder == nil {
		encoder = json.NewEncoder(os.Stdout)
	}
	return encoder
}

func defaultL3Callback() func(device *devices.L3Device, event devices.L3DeviceEvent) {
	encoder = getEncoder()
	return func(device *devices.L3Device, event devices.L3DeviceEvent) {
		t := make(map[string]interface{})
		t["event"] = event.String()
		t["index"] = device.Index
		t["namespace"] = device.Namespace
		t["addresses"] = device.IP
		encoder.Encode(t)
	}
}

func defaultNSCallback() func(namespace *devices.Namespace, event devices.NSEvent) {
	encoder := getEncoder()

	getKeys := func(m map[string]string) []string {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		return keys
	}

	getRoutes := func(routes []netlink.Route) []map[string]string {
		t := make([]map[string]string, len(routes))
		for i, r := range routes {
			m := make(map[string]string)
			if r.Src != nil {
				m["source"] = fmt.Sprintf("%s", r.Src)
			}
			if r.Dst != nil {
				m["destination"] = fmt.Sprintf("%s", r.Dst)
			}
			if r.Gw != nil {
				m["gateway"] = fmt.Sprintf("%s", r.Gw)
			}
			if r.NewDst != nil {
				m["nexthop"] = fmt.Sprintf("Dst: %s", r.NewDst)
			}
			t[i] = m
		}
		return t
	}

	return func(namespace *devices.Namespace, change devices.NSEvent) {
		t := make(map[string]interface{})
		t["event"] = change.String()
		t["name"] = namespace.Name
		switch change {
		case devices.NSTypeChange:
			t["type"] = namespace.Type
		case devices.NSConnect:
			t["connections"] = getKeys(namespace.Connections)
		case devices.NSDisconnect:
			t["connections"] = getKeys(namespace.Connections)
		case devices.NSRouteAdd:
			t["route"] = getRoutes(namespace.Routes)
		case devices.NSRouteDelete:
			t["route"] = getRoutes(namespace.Routes)
		}
		encoder.Encode(t)
	}
}

func defaultVethCallback() func(veth *devices.Veth, events devices.VethEvent) {
	encoder := getEncoder()
	return func(veth *devices.Veth, event devices.VethEvent) {
		t := make(map[string]interface{})
		t["event"] = event.String()
		t["name"] = veth.Name
		t["namespace"] = veth.Namespace
		t["index"] = veth.Index
		switch event {
		case devices.VethPair:
			t["peerName"] = veth.PeerName
			t["peerNamespace"] = veth.PeerNamespace
			t["peerIndex"] = veth.PeerIndex
		}
		encoder.Encode(t)
	}
}

func main() {
	fmt.Println("Hello OpenVNV")
	consoleDisplay = flag.Bool("events", false, "Use -events to display events on console")
	dumpIP = flag.String("ip", "empty", "Use -ip=<ip>:<port> to send events to remote tcp connection")
	flag.Parse()
	var sock io.Writer
	if *dumpIP != "empty" {
		var err error
		sock, err = net.Dial("tcp", *dumpIP)
		if err != nil {
			fmt.Println("ERROR: CONNECTING TCP SERVER", err)
			os.Exit(1)
		}
	} else {
		sock = os.Stdout
	}
	devices.SetWriter(sock)

	if *consoleDisplay {
		fmt.Println("Console Display Enabled")
		n := defaultNSCallback()
		v := defaultVethCallback()
		d := defaultL3Callback()
		devices.SubscribeAllNamespaceEvents(n)
		devices.SubscribeAllVethEvents(v)
		devices.SubscribeAllL3DeviceEvents(d)
	}
	createExistingNamespaces(*consoleDisplay)
	go netnsTopoligy()
	dumpTopology()
}

func netnsTopoligy() {
	var netnsCreateChannel = make(chan string)
	var netnsDestroyChannel = make(chan string)
	var errChan = make(chan error)
	go SubscribeDockerNetnsUpdate(&netnsCreateChannel, &netnsDestroyChannel, errChan)
	for {
		select {
		case u := <-netnsCreateChannel:
			fmt.Println("GOT:", u)
			processNewNamespace(u, *consoleDisplay)
		case u := <-netnsDestroyChannel:
			topology.DeleteNamespace(u)
		case err := <-errChan:
			fmt.Println("ERROR: SUBSCRIBEDOCKERUPDATE", err)
		}
	}
}

func processNewNamespace(name string, consoleDisplay bool) {
	targetNS, err := netns.GetFromDocker(name)
	if err != nil {
		fmt.Println("ERROR: GETTING DOCKER NS: ", err, namespace)
		return
	}
	runtime.LockOSThread()

	defaultNS, err := netns.Get()
	if err != nil {
		fmt.Println("ERROR: GETTING CURRENT NS: ", err, namespace)
		return
	}

	err = netns.Set(targetNS)
	t := topology.CreateNamespace(name, &targetNS)
	if err != nil {
		fmt.Println("ERROR: SETTING GOROUTINE TO DOCKER NS: ", err, namespace)
		return
	}
	createDevices(t, consoleDisplay)
	err = netns.Set(defaultNS)
	if err != nil {
		fmt.Println("ERROR: SETTING GOROUTINE TO DEFAULT NS: ", err, namespace)
		return
	}

	runtime.UnlockOSThread()
	go listenOnLinkMessagesWithExisting(t, &targetNS, consoleDisplay)
}

func dumpTopology() {
	commands := "Enter:\nIndex Number to look for device state or\n'*' to look for all devices\n'bye' to exit\n'help' to print this message again"
	fmt.Println(commands)
	reader := bufio.NewReader(os.Stdin)
	for {
		text, err := reader.ReadString('\n')
		text = strings.Trim(text, "\n")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if text == "bye" {
			for ns, _ := range topology.Namespaces {
				time.After(5 * time.Second)
				topology.DeleteNamespace(ns)
			}
			os.Exit(0)
		} else if text == "*" {
			for ns, n := range topology.Namespaces {
				fmt.Println("\nDevices for namespace", ns)
				fmt.Println(n.Routes)
				n.DumpAll()
			}
		} else if text == "help" {
			fmt.Println("\n\n", commands)
		} else {
			namespace = topology.GetDefaultNamespace()
			if index, err := strconv.ParseInt(text, 10, 64); err == nil {
				namespace.Dump(int(index))
			}
		}
	}
}
