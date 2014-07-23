package main

import "fmt"
import "io"
import "net"
import "os"


func runServer() {
  root, err := os.Getwd()
	checkError(err)

	fmt.Println("Zync server starting...")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	checkError(err)

	fmt.Printf("Zync server started on port %d.\n", port)
	for {
		conn, err := listener.Accept()
		checkError(err)
		defer conn.Close()
		handleConnection(conn, root)
		fmt.Println("Client disconnected.")
	}
}

func handleConnection(conn net.Conn, root string) {
	// Server cuts off client on any error, but continues running.
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "Disconnecting client abnormally.")
		}
	}()

	fmt.Println("Client connected:", conn.RemoteAddr())

	version, err := expectVersion(conn)
	checkError(err)

	fmt.Println("Client requested protocol version:", version)
	if version != ProtoVersion {
		// Exact match on version is required (currently).
		checkError(send(conn, false))
		return
	} else {
		checkError(send(conn, true))
	}

  files := enumerateFiles(root)
  for {
    cmd, err := expectCommand(conn)
    if err == io.EOF {
      return
    }

    checkError(err)

    switch(cmd) {
    case CmdRequestNextFileInfo:
      fi, ok := <-files
      if ok {
        checkError(send(conn, true))
        checkError(send(conn, fi))
      } else {
        checkError(send(conn, false))
      }
    default:
      panic(fmt.Errorf("Unrecognized client command: %d", cmd))
    }
  }
}
