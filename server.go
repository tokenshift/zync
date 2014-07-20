package main

import "fmt"
import "net"
import "os"

func runServer() {
	fmt.Println("Zync server starting...")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	checkError(err)

	fmt.Printf("Zync server started on port %d.\n", port)
	for {
		conn, err := listener.Accept()
		checkError(err)
		defer conn.Close()
		handleConnection(conn)
		fmt.Println("Client disconnected.")
	}
}

func handleConnection(conn net.Conn) {
	// Server cuts off client on any error, but continues running.
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "Disconnecting client abnormally.")
		}
	}()

	fmt.Println("Client connected:", conn.RemoteAddr())

	version, err := recvVersion(conn)
	checkError(err)

	fmt.Println("Client requested protocol version:", version)
	if version != ProtoVersion {
		// Exact match on version is required (currently).
		sendBool(conn, false)
		return
	} else {
		checkError(sendBool(conn, true))
	}
}
