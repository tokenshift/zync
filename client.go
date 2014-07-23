package main

import "fmt"
import "io"
import "net"
import "os"
import "path"
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
	checkError(send(conn, ProtoVersion))
	accepted, err := expectBool(conn)
	checkError(err)
	if !accepted {
		fmt.Fprintln(os.Stderr, "Server rejected protocol version", ProtoVersion)
		os.Exit(1)
	}

	// Synchronization process:
	// 1. Client asks server for the next file it sees. Server returns filename
	// and hash.
	// 2. Client compares the server's file (by name) to its own.
	// 3. If server's file is 'before' (lexically) the client's file, client
	// requests and receives server's file, saving it to disk at the correct
	// location.
	// 4. If server's file is 'after' the client's file, client sends all of its
	// files 'up to' that file to the server.
	// 5. If the filenames match, filesizes and modification times are used to
	// check if the files are different.
	// 6. If the files are different, use the chosen conflict resolution
	// mechanism to determine which side 'wins'; the client either requests the
	// file from the server or sends its own file to the server.
	myFiles := enumerateFiles(root)

	myNext, myAny := <-myFiles
	svrNext, svrAny := requestNextFileInfo(conn)
	for myAny || svrAny {
		if svrAny && (!myAny || svrNext.Path < myNext.Path) {
			requestAndCreateFile(conn, root, svrNext)
			svrNext, svrAny = requestNextFileInfo(conn)
		} else if myAny && (!svrAny || svrNext.Path > myNext.Path) {
			fmt.Println("TODO: Send", myNext, "to server")
			myNext, myAny = <-myFiles
		} else {
			fmt.Println("TODO: Compare file info for", myNext)
			myNext, myAny = <-myFiles
			svrNext, svrAny = requestNextFileInfo(conn)
		}
	}
}

// Requests the specified file from the server, and saves it to the relevant
// location on disk.
func requestAndCreateFile(conn net.Conn, root string, fi FileInfo) {
	abs := path.Join(root, fi.Path)

	// If this is a folder, just go ahead and create it; no need to ask the
	// server for anything.
	if fi.IsDir {
		fmt.Println("Creating folder", fi.Path)
		checkError(os.Mkdir(abs, os.ModeDir | fi.Mode))
		return
	}

	fmt.Println("Requesting", fi.Path, "from server.")
	checkError(send(conn, FileRequest { Path: fi.Path }))
	yes, err := expectBool(conn)
	checkError(err)

	if yes {
		msgType, err := recvMessageType(conn)
		checkError(err)
		if msgType != MsgFile {
			panic(fmt.Errorf("Expected MsgFile (%v), got %v", MsgFile, msgType))
		}

		path, err := expectString(conn)
		checkError(err)
		if path != fi.Path {
			panic(fmt.Errorf("Requested %v, server provided %v", fi.Path, path))
		}

		size, err := expectInt64(conn)
		checkError(err)
		if size > MaxFileSize {
			panic(fmt.Errorf("File too large: %d bytes", size))
		}

		file, err := os.Create(abs)
		checkError(err)

		written, err := io.CopyN(file, conn, size)
		checkError(err)

		if written != size {
			panic(fmt.Errorf("Failed to receive full contents of %s (%d bytes)", fi.Path, size))
		}

		checkError(checkMessageTerminator(conn))
	} else {
		fmt.Fprintln(os.Stderr, "WARNING: Server refused to provide", fi.Path)
	}
}

// Asks the server for and receives the next file that it sees.
func requestNextFileInfo(conn net.Conn) (FileInfo, bool) {
	checkError(send(conn, CmdRequestNextFileInfo))
	yes, err := expectBool(conn)
	checkError(err)

	if yes {
		fi, err := expectFileInfo(conn)
		checkError(err)
		return fi, true
	} else {
		return FileInfo{}, false
	}
}
