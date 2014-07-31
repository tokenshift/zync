package main

import "fmt"
import "os"
import "strconv"

func main() {
	args := os.Args

	// Determine the run mode.
	server, args := argFlag(args, "server", "s")
	client, connectUri, args := argOption(args, "connect", "c")

	if server && client {
		fmt.Fprintln(os.Stderr, "Only one of --connect (-c), --server (-s) can be specified.")
		os.Exit(1)
	}

	// Global options.
	hash, args = argFlag(args, "hash", "h")
	interactive, args = argFlag(args, "interactive", "i")
	verbose, args = argFlag(args, "verbose", "v")

	if server {
		// Server mode.
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
	} else if client {
		// Client mode.
		if connectUri == "" {
			fmt.Fprintln(os.Stderr, "--connect (-c) requires a URI.")
			os.Exit(1)
		}

		_, keepWhose, args = argOption(args, "keep", "k")
		if keepWhose != "" && keepWhose != "theirs" && keepWhose != "mine" {
			fmt.Fprintln(os.Stderr, "--keep (-k) must be 'theirs' or 'mine'.")
			os.Exit(1)
		}

		autoDelete, args = argFlag(args, "delete", "d")
		if autoDelete && keepWhose == "" {
			fmt.Fprintln(os.Stderr, "--delete (-d) can only be used in combination with --keep (-k).")
			os.Exit(1)
		}

		reverse, args = argFlag(args, "reverse", "r")

		runClient(connectUri)
	} else {
		fmt.Fprintln(os.Stderr, "One of --connect (-c), --server (-s) must be specified.")
	}
}

// Simple error handling function. Prints the error to STDOUT and panics.
func checkError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		panic(err)
	}
}

func assert(b bool, msg string) {
	if !b {
		panic(fmt.Errorf(msg))
	}
}
