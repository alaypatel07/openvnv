package main

import (
	"fmt"
	"github.com/alaypatel07/openvnv/devices"
	"bufio"
	"os"
	"strconv"
	"strings"
	"flag"
	"net"
	"io"
	"github.com/vishvananda/netns"
	"runtime"
	"encoding/json"
)

var namespace *devices.Namespace
var topology = devices.NewTopology()
var consoleDisplay *bool

var dumpIP *string

func main() {
	fmt.Println("Hello OpenVNV")
	consoleDisplay = flag.Bool("events", false, "Use -events to display events on console")
	dumpIP = flag.String("ip", "empty", "Use -ip=<ip>:<port> to send events to remote tcp connection")
	flag.Parse()
	if *consoleDisplay {
		fmt.Println("Console Display Enabled")
		defaultNSCallback := func(namespace devices.Namespace, change devices.NSEvent) {
			//fmt.Println("Namespace:", namespace.Name, "Event", int(change), "occured which means", change)
			t := make(map[string]interface{})
			t["event"] = change.String()
			t["data"] = namespace
			json.NewEncoder(os.Stdout).Encode(t)
		}
		devices.SubscribeAllNamespaceEvents(defaultNSCallback)
	}
	createExistingNamespaces(*consoleDisplay)
	go netnsTopoligy()
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
	dumpTopology()
}

func netnsTopoligy() {
	var netnsCreateChannel = make(chan string)
	var netnsDestroyChannel = make(chan string)
	var errChan = make(chan error)
	go SubscribeDockerNetnsUpdate(&netnsCreateChannel, &netnsDestroyChannel, errChan);
	for {
		select {
		case u := <-netnsCreateChannel:
			processNewNamespace(u, *consoleDisplay)
		case u := <- netnsDestroyChannel:
			topology.DeleteNamespace(u)
		case err := <- errChan:
			fmt.Println("ERROR: SUBSCRIBEDOCKERUPDATE", err)
		}
	}
}

func processNewNamespace(name string, consoleDisplay bool) {
	t := topology.CreateNamespace(name)
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
			for ns, _ := range topology {
				topology.DeleteNamespace(ns)
			}
			os.Exit(0)
		} else if text == "*" {
			for ns, n := range topology {
				fmt.Println("\nDevices for namespace", ns)
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

