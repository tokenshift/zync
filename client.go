package main

import "fmt"
import "os"
import "net"
import "regexp"

var portRx = regexp.MustCompile(":\\d+$")

func runClient(connectUri string) {
	root, err := os.Getwd()
	checkError(err)

	match := portRx.FindString(connectUri)
	if match == "" {
		connectUri = fmt.Sprintf("%s:%d", connectUri, port)
	}

	fmt.Println("Starting Zync client.")
	fmt.Printf("Working directory is %v.\n", root)

	fmt.Printf("Connecting to Zync server at %s...\n", connectUri)
	_, err = net.Dial("tcp", connectUri)
	checkError(err)
}
