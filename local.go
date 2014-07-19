package main

import "fmt"
import "os"
import "net"
import "net/url"
import "regexp"

var portRx = regexp.MustCompile(":\\d+$")

func runLocal(connectUri string) {
	root, err := os.Getwd()
	checkError(err)

	uri, err := url.Parse(connectUri)
	checkError(err)

	if uri.Scheme != "zync" && uri.Scheme != "file" {
		fmt.Fprintf(os.Stdout, "Unsupported scheme: '%s'. Only 'zync' and 'file' are supported.\n", uri.Scheme)
		os.Exit(1)
	}

	host := uri.Host
	match := portRx.FindString(host)
	if match == "" {
		host = fmt.Sprintf("%s:%d", host, port)
	}

	fmt.Println("Starting local Zync node.")
	fmt.Printf("Working directory is %v.\n", root)

	fmt.Printf("Connecting to Zync node at %s...\n", host)
	_, err = net.Dial("tcp", host)
	checkError(err)
}
