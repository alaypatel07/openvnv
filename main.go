package main

import (
	"fmt"
	"github.com/alaypatel07/openvnv/devices"
	"bufio"
	"os"
	"strconv"
	"strings"
	"flag"
	"github.com/vishvananda/netns"
	"net"
	"io"
)

var namespace *devices.Namespace
var topology *devices.Topology
var consoleDisplay *bool

var dumpIP *string

func main() {
	fmt.Println("Hello OpenVNV")
	go netnsTopoligy()
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
		defaultNSCallback := func(namespace devices.Namespace, change devices.NSEvent) {
			fmt.Println("Namespace:", namespace.Name, "Event", int(change), "occured which means", change)
		}
		devices.SubscribeAllNamespaceEvents(defaultNSCallback)
	}
	go dumpTopology()
	currNs, err := netns.Get()
	if err != nil {
		fmt.Println("ERROR: GETTING CURRENT NAMESPACE", err)
		os.Exit(1)
	}
	n := NewNamespace("default", &currNs)
	listenOnLinkMessagesWithExisting(&n, *consoleDisplay)
}

func netnsTopoligy() {
	var netnsCreateChannel = make(chan Namespace)
	var netnsDestroyChannel = make(chan Namespace)
	var errChan = make(chan error)
	go SubscribeDockerNetnsUpdate(&netnsCreateChannel, &netnsDestroyChannel, errChan);
	for {
		select {
		case u := <-netnsCreateChannel:
			go listenOnLinkMessagesWithExisting(&u, *consoleDisplay)
		case u := <- netnsDestroyChannel:
			devices.DeleteNamespace(u.name)
			*u.doneChannel <- true
		case err := <- errChan:
			fmt.Println("ERROR: SUBSCRIBEDOCKERUPDATE", err)
		}
	}
}

func dumpTopology() {
	namespace = devices.GetDefaultNamespace()
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
			os.Exit(0)
		} else if text == "*" {
			t := devices.GetTopology()
			for ns, n := range t {
				fmt.Println("\nDevices for namespace", ns)
				n.DumpAll()
			}
		} else if text == "help" {
			fmt.Println("\n\n", commands)
		} else {
			if index, err := strconv.ParseInt(text, 10, 64); err == nil {
				namespace.Dump(int(index))
			}
		}
	}
}

