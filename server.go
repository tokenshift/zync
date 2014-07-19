package main

import "fmt"
import "net"

func runServer() {
	fmt.Println("Zync server starting...")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	checkError(err)

	fmt.Printf("Zync server started on port %d.\n", port)
	for {
		conn, err := listener.Accept()
		checkError(err)
		handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	fmt.Println("Client connected")
}
