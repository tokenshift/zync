package main

import "fmt"
import "io"
import "net"
import "os"
import "path"

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
		msg, msgType, err := recv(conn)
		if err == io.EOF {
			return
		}

		checkError(err)

		switch msgType {
		case MsgCommand:
			switch msg.(Command) {
			case CmdRequestNextFileInfo:
				handleCmdRequestNextFileInfo(conn, files)
			default:
				panic(fmt.Errorf("Unrecognized command: %d", msg))
			}
		case MsgFileRequest:
			handleMsgFileRequest(conn, root, msg.(FileRequest))
		default:
			panic(fmt.Errorf("Unrecognized message type: %d", msgType))
		}
	}
}

func handleCmdRequestNextFileInfo(conn net.Conn, files <-chan FileInfo) {
	fi, ok := <-files
	if ok {
		checkError(send(conn, true))
		checkError(send(conn, fi))
	} else {
		checkError(send(conn, false))
	}
}

var fileBuffer = make([]byte, 1024 * 1024)
func handleMsgFileRequest(conn net.Conn, root string, req FileRequest) {
	fmt.Println("Client requested", req.Path)

	abs := path.Join(root, req.Path)
	if fStat, err := os.Stat(abs); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "WARNING: Client requested nonexistant file", req.Path)
		checkError(send(conn, false))
		return
	} else if file, err := os.Open(abs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		checkError(send(conn, false))
		return
	} else {
		fmt.Println("Sending", req.Path, "to client.")
		checkError(send(conn, true))

		// Sending a file follows the same structure as other messages (type, data,
		// terminator), but is handled separately to avoid a recipient
		// 'accidentally' receiving a potentially very large message that they were
		// not expecting.
		checkError(writeMessageType(conn, MsgFile))
		checkError(send(conn, req.Path))
		checkError(send(conn, fStat.Size()))

		sent, err := io.Copy(conn, file)
		checkError(err)
		if sent != fStat.Size() {
			panic(fmt.Errorf("Failed to send full contents of %s (%d bytes)", req.Path, fStat.Size()))
		}

		checkError(writeInt32(conn, MessageTerminator))
	}
}
