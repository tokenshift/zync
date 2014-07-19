package main

import "fmt"
import "os"
import "strconv"

func main() {
	args := os.Args

	// Determine the run mode.
	daemon, args := argFlag(args, "daemon", "d")
	connect, connectUri, args := argOption(args, "connect", "c")

	if daemon && connect {
		fmt.Fprintln(os.Stderr, "Only one of --connect (-c), --daemon (-d) can be specified.")
		os.Exit(1)
	}

	// Global options.
	hash, args = argFlag(args, "hash", "h")
	interactive, args = argFlag(args, "interactive", "i")
	verbose, args = argFlag(args, "verbose", "v")

	if daemon {
		// Daemon mode.
		portSpecified, portStr, _ := argOption(args, "port", "p")
		if portSpecified {
			portNum, err := strconv.ParseInt(portStr, 10, 0)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Port must be a number")
				os.Exit(1)
			}
			port = int(portNum)
		}

		runServer()
	} else if connect {
		if connectUri == "" {
			fmt.Fprintln(os.Stderr, "--connect (-c) requires a URI.")
			os.Exit(1)
		}

		// Local mode.
		_, keepWhose, args = argOption(args, "keep", "k")
		autoDelete, args = argFlag(args, "delete", "d")
		reverse, args = argFlag(args, "reverse", "r")

		runClient(connectUri)
	} else {
		fmt.Fprintln(os.Stderr, "One of --connect (-c), --daemon (-d) must be specified.")
	}
}

// Simple error handling function. Prints the error to STDOUT and exits.
func checkError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
