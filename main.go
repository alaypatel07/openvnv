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

func defaultL3Callback() func(device *devices.L3Device, event devices.L3DeviceEvent) {
	encoder = devices.GetEncoder()
	return func(device *devices.L3Device, event devices.L3DeviceEvent) {
		t := make(map[string]interface{})
		t["name"] = device.Index
		t["namespace"] = device.Namespace
		t["addresses"] = device.IP
		t["connections"] = device.L2EventChannel().Master
		t["indexName"] = "device1"
		switch event {
		case devices.L3DeviceCreate:
			t["event"] = "create"
		case devices.L3DeviceDelete:
			t["event"] = "delete"
		default:
			t["event"] = "update"
		}
		encoder.Encode(t)
	}
}

func defaultNSCallback() func(namespace *devices.Namespace, event devices.NSEvent) {
	encoder := devices.GetEncoder()

	getKeys := func(name string, m map[string]string) []string {
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
		t["name"] = namespace.Name
		t["indexName"] = "namespace1"
		t["connection"] = getKeys(namespace.Name, namespace.Connections)
		t["route"] = getRoutes(namespace.Routes)
		t["mode"] = namespace.Type
		switch change {
		case devices.NSCreate:
			t["event"] = "create"
		case devices.NSTypeChange:
			t["event"] = "update"
		case devices.NSConnect:
			t["event"] = "update"
		case devices.NSDisconnect:
			t["event"] = "update"
		case devices.NSRouteAdd:
			t["event"] = "update"
		case devices.NSRouteDelete:
			t["event"] = "update"
		}
		encoder.Encode(t)
	}
}

func defaultVethCallback() func(veth *devices.Veth, events devices.VethEvent) {
	encoder := devices.GetEncoder()
	return func(veth *devices.Veth, event devices.VethEvent) {
		t := make(map[string]interface{})
		t["event"] = event.String()
		t["name"] = veth.Name
		t["indexName"] = "device1"
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
	go registerWS(":8080")
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
		nws := defaultNSWSCallback()
		devices.SubscribeAllNamespaceEvents(n)
		devices.SubscribeAllVethEvents(v)
		devices.SubscribeAllL3DeviceEvents(d)
		devices.SubscribeAllNamespaceEvents(nws)
	}
	createExistingNamespaces(*consoleDisplay)
	go netnsTopoligy()
	dumpTopology()
}
func defaultNSWSCallback() func(namespace *devices.Namespace, event devices.NSEvent) {
	getKeys := func(m map[string]string) []string {
		keys := make([]string, 0, len(m))
		for k := range m {
			z := strings.Split(k, ":")
			keys = append(keys, z[0])
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

	callback := func(namespace *devices.Namespace, event devices.NSEvent) {
		//fmt.Println("NSWS Called", event, namespace.Routes, namespace.Connections)
		temp := make(map[string]interface{})
		temp["name"] = namespace.Name
		temp["peer"] = getKeys(namespace.Connections)
		temp["route"] = getRoutes(namespace.Routes)
		temp["mode"] = namespace.Type
		e := WsEvents{
			DeviceType: "namespace",
			EventData:  temp,
			EventType:  event.String(),
		}
		chans := GetChannels()

		for _, value := range *chans {
			//fmt.Println("Sending to", s)
			*value <- e
		}
	}
	return callback
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
				fmt.Println("\nConnections for namespace", ns)
				fmt.Println(n.Connections)
				fmt.Println("\nRoutes for namespace", ns)
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
