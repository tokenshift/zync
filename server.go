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
		case MsgFileDeletionRequest:
			handleMsgFileDeletionRequest(conn, root, msg.(FileDeletionRequest))
		case MsgFileOffer:
			handleMsgFileOffer(conn, root, msg.(FileOffer))
		case MsgFileRequest:
			handleMsgFileRequest(conn, root, msg.(FileRequest))
		default:
			panic(fmt.Errorf("Unrecognized message type: %d", msgType))
		}
	}
}

var lastSentFilePath string

func handleCmdRequestNextFileInfo(conn net.Conn, files <-chan FileInfo) {
	fi, ok := <-files
	if ok {
		checkError(send(conn, true))
		checkError(send(conn, fi))
		lastSentFilePath = fi.Path
	} else {
		checkError(send(conn, false))
	}
}

func handleMsgFileDeletionRequest(conn net.Conn, root string, req FileDeletionRequest) {
	logVerbose("Client requested deletion of", req.Path)

	if restrict || restrictAll {
		// Server was run with the --restrict (-r) or --Restrict (-R) option;
		// refuse to delete any file.
		checkError(send(conn, false))
	} else if lastSentFilePath != req.Path {
		// Refuse to delete the file if it isn't the last file that the server
		// informed the client of. Otherwise, the client could be trying
		// something sneaky...
		checkError(send(conn, false))
	} else {
		// Delete the local file.
		checkError(send(conn, true))
		deleteLocalFile(root, req.Path)
	}
}

var fileBuffer = make([]byte, 1024 * 1024)
func handleMsgFileRequest(conn net.Conn, root string, req FileRequest) {
	logVerbose("Client requested", req.Path)

	abs := path.Join(root, req.Path)
	if fStat, err := os.Stat(abs); os.IsNotExist(err) {
		logWarning("Client requested nonexistant file", req.Path)
		checkError(send(conn, false))
	} else {
		logInfo("Sending", req.Path, "to client.")
		checkError(send(conn, true))

		fi, err := fileInfo(root, abs, fStat)
		checkError(err)
		checkError(sendFile(conn, fi, abs))
	}
}

func handleMsgFileOffer(conn net.Conn, root string, offer FileOffer) {
	path := path.Join(root, offer.Info.Path)

	_, err := os.Stat(path)
	if restrictAll && !os.IsNotExist(err) {
		// Refuse the offer; server was run in --Restrict (-R) mode.
		logVerbose("Rejecting client's", offer.Info.Path)
		checkError(send(conn, false))
	} else if offer.Info.IsDir {
		// Reject the offer, create the folder directly.
		logVerbose("Creating folder", offer.Info.Path)
		checkError(os.Mkdir(path, os.ModeDir | offer.Info.Mode))
		checkError(send(conn, false))
	} else {
		// Accept the offer.
		checkError(send(conn, true))

		// Receive the file.
		logInfo("Receiving", offer.Info.Path, "from client.")
		checkError(recvFile(conn, offer.Info, path, true))
	}
}
