package main

import "bufio"
import "fmt"
import "net"
import "os"
import "path/filepath"
import "regexp"
import "strings"
import "time"

var portRx = regexp.MustCompile(":\\d+$")

type ConflictType int
const (
	// Different versions of the file exist on the server and the client.
	Conflict ConflictType = iota

	// The client is missing a file that the server has.
	Missing

	// The server is missing a file that the client has.
	New
)

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

	logInfo("Starting Zync client.")
	logInfo("Working directory is", root)

	logInfo("Connecting to Zync server at", connectUri)
	conn, err := net.Dial("tcp", connectUri)
	checkError(err)
	defer conn.Close()

	// Version Check
	checkError(send(conn, ProtoVersion))
	accepted, err := expectBool(conn)
	checkError(err)
	if !accepted {
		logError("Server rejected protocol version", ProtoVersion)
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
			if interactive {
				promptForAction(conn, root, Missing, svrNext, myNext)
			} else if keepWhose == "mine" && autoDelete {
				requestFileDeletion(conn, svrNext.Path)
			} else {
				requestAndSaveFile(conn, root, svrNext, false)
			}
			svrNext, svrAny = requestNextFileInfo(conn)
		} else if myAny && (!svrAny || svrNext.Path > myNext.Path) {
			if interactive {
				promptForAction(conn, root, New, svrNext, myNext)
			} else if keepWhose == "theirs" && autoDelete {
				deleteLocalFile(root, myNext.Path)
			} else {
				offerAndSendFile(conn, root, myNext)
			}
			myNext, myAny = <-myFiles
		} else {
			resolve(conn, root, myNext, svrNext)
			myNext, myAny = <-myFiles
			svrNext, svrAny = requestNextFileInfo(conn)
		}
	}

	logInfo("Complete, disconnecting.")
}

func resolve(conn net.Conn, root string, mine FileInfo, theirs FileInfo) {
	assert(mine.Path == theirs.Path, "Cannot resolve differing paths.")

	if mine.IsDir || theirs.IsDir {
		if mine.IsDir != theirs.IsDir {
			logError("Tree conflict at", mine.Path)
		}
		return
	}

	logVerbose("Comparing", mine.Path)
	if mine.Size == theirs.Size && mine.ModTime.Equal(theirs.ModTime) {
		logVerbose("Files match, skipping.")
		return
	}

	if interactive {
		promptForAction(conn, root, Conflict, theirs, mine)
	} else if keepWhose == "mine" || (keepWhose == "" && mine.ModTime.After(theirs.ModTime)) {
		// Use the client's version.
		logVerbose("Sending", mine.Path, "to server.")
		offerAndSendFile(conn, root, mine)
	} else if keepWhose == "theirs" || (keepWhose == "" && theirs.ModTime.After(mine.ModTime)) {
		// Use the server's version.
		logVerbose("Requesting", theirs.Path, "from server.")
		requestAndSaveFile(conn, root, theirs, true)
	} else {
		// Could not automatically resolve.
		logWarning("Failed to resolve", mine.Path, "automatically; mod times match.")
	}
}

// Asks the user what action should be taken for a specific file.
func promptForAction(conn net.Conn, root string, ct ConflictType, theirs, mine FileInfo) {
	switch (ct) {
	case Conflict:
		fmt.Println("CONFLICT:", mine.Path)

		fmt.Printf("Server has: %d bytes ", theirs.Size)
		if theirs.Size > mine.Size {
			fmt.Print("(bigger)")
		} else if theirs.Size < mine.Size {
			fmt.Print("(smaller)")
		} else {
			fmt.Print("(same)")
		}
		fmt.Printf(", %s ", theirs.ModTime.Format(time.RFC3339))
		if theirs.ModTime.After(mine.ModTime) {
			fmt.Println("(newer)")
		} else if theirs.ModTime.Before(mine.ModTime) {
			fmt.Println("(older)")
		} else {
			fmt.Println("(same)")
		}

		fmt.Printf("Client has: %d bytes ", mine.Size)
		if theirs.Size > mine.Size {
			fmt.Print("(smaller)")
		} else if theirs.Size < mine.Size {
			fmt.Print("(bigger)")
		} else {
			fmt.Print("(same)")
		}
		fmt.Printf(", %s ", theirs.ModTime.Format(time.RFC3339))
		if theirs.ModTime.After(mine.ModTime) {
			fmt.Println("(older)")
		} else if theirs.ModTime.Before(mine.ModTime) {
			fmt.Println("(newer)")
		} else {
			fmt.Println("(same)")
		}

		action := requestUserInput("Action ([g]ive mine, [a]ccept theirs, [s]kip)",
			keepWhose, "give", "accept", "skip")

		switch action {
		case "give":
			logVerbose("Sending", mine.Path, "to server.")
			offerAndSendFile(conn, root, mine)
		case "accept":
			logVerbose("Requesting", theirs.Path, "from server.")
			requestAndSaveFile(conn, root, theirs, true)
		case "skip":
			logVerbose("Skipping", mine.Path)
		}
	case Missing:
		fmt.Println("MISSING:", theirs.Path)
		dflt := "accept"
		if keepWhose == "mine" && autoDelete {
			dflt = "delete"
		}
		action := requestUserInput("Action ([a]ccept theirs, [d]elete theirs, [s]kip)",
			dflt, "accept", "delete", "skip")
		switch action {
		case "accept":
			logVerbose("Requesting", theirs.Path, "from server.")
			requestAndSaveFile(conn, root, theirs, true)
		case "delete":
			requestFileDeletion(conn, theirs.Path)
		case "skip":
			logVerbose("Skipping", theirs.Path)
		}
	case New:
		fmt.Println("NEW:", mine.Path)
		dflt := "give"
		if keepWhose == "theirs" && autoDelete {
			dflt = "delete"
		}
		action := requestUserInput("Action ([g]ive mine, [d]elete mine, [s]kip)",
			dflt, "give", "delete", "skip")
		switch action {
		case "give":
			logVerbose("Sending", mine.Path, "to server.")
			offerAndSendFile(conn, root, mine)
		case "delete":
			deleteLocalFile(root, mine.Path)
		case "skip":
			logVerbose("Skipping", mine.Path)
		}
	}
}

// Helper function to request and parse user input."
func requestUserInput(prompt, dflt string, options...string) string {
	input := bufio.NewReader(os.Stdin)

	for {
		if dflt == "" {
			fmt.Printf("%s: ", prompt)
		} else {
			fmt.Printf("%s: [%s] ", prompt, dflt[0:1])
		}

		line, err := input.ReadString('\n')
		checkError(err)

		line = strings.TrimSpace(line)

		for _, opt := range(options) {
			if line == opt || line[0] == opt[0] {
				return opt
			}
		}

		fmt.Println("Invalid input: %s", line)
	}
}

// Deletes the client's version of a file that has been deleted on the server.
func deleteLocalFile(root, name string) {
	logVerbose("Deleting", name)
	checkError(os.RemoveAll(filepath.Join(root, name)))
}

// Asks the server to delete their version of a file that has been deleted on
// the client.
func requestFileDeletion(conn net.Conn, path string) {
	logVerbose("Asking server to delete", path)
	checkError(send(conn, FileDeletionRequest { Path: path }))

	yes, err := expectBool(conn)
	checkError(err)

	if !yes {
		logWarning("Server refused to delete", path)
	}
}

// Requests the specified file from the server, and saves it to the relevant
// location on disk.
func requestAndSaveFile(conn net.Conn, root string, fi FileInfo, overwrite bool) {
	abs := filepath.Join(root, fi.Path)

	// If this is a folder, just go ahead and create it; no need to ask the
	// server for anything.
	if fi.IsDir {
		logVerbose("Creating folder", fi.Path)
		checkError(os.Mkdir(abs, os.ModeDir | fi.Mode))
		return
	}

	logInfo("Requesting", fi.Path, "from server.")
	checkError(send(conn, FileRequest { Path: fi.Path }))
	yes, err := expectBool(conn)
	checkError(err)

	if yes {
		logVerbose("Receiving", fi.Path, "from server.")
		checkError(recvFile(conn, fi, abs, overwrite))
	} else {
		logWarning("Server refused to provide", fi.Path)
	}
}

// Offers a file to the server and sends it if the server accepts.
func offerAndSendFile(conn net.Conn, root string, fi FileInfo) {
	logVerbose("Offering", fi.Path, "to server.")
	checkError(send(conn, FileOffer { Info: fi }))

	yes, err := expectBool(conn)
	checkError(err)

	if yes {
		logInfo("Sending", fi.Path, "to server.")
		path := filepath.Join(root, fi.Path)
		checkError(sendFile(conn, fi, path))
	} else {
		logVerbose("Server refused to accept", fi.Path)
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
