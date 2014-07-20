package main

import "fmt"
import "os"
import "net"
import "regexp"

var portRx = regexp.MustCompile(":\\d+$")

func runClient(connectUri string) {
	// Client bails on any error.
	defer func() {
		if err := recover(); err != nil {
			os.Exit(1)
		}
	}()

	root, err := os.Getwd()
	checkError(err)

	match := portRx.FindString(connectUri)
	if match == "" {
		connectUri = fmt.Sprintf("%s:%d", connectUri, port)
	}

	fmt.Println("Starting Zync client.")
	fmt.Printf("Working directory is %v.\n", root)

	fmt.Printf("Connecting to Zync server at %s...\n", connectUri)
	conn, err := net.Dial("tcp", connectUri)
	checkError(err)

	// Version Check
	checkError(sendVersion(conn, ProtoVersion))
	accepted, err := recvBool(conn)
	checkError(err)
	if !accepted {
		fmt.Fprintln(os.Stderr, "Server rejected protocol version", ProtoVersion)
		os.Exit(1)
	}
}
