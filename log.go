package main

import "fmt"
import "io"
import "os"

func log(out io.Writer, prefix string, args ...interface{}) {
	if prefix != "" {
		args = append([]interface{} { prefix }, args...)
	}
	fmt.Fprintln(out, args...)
}

func logError(args ...interface{}) {
	log(os.Stderr, "ERROR:", args...)
}

func logInfo(args ...interface{}) {
	log(os.Stdout, "", args...)
}

func logVerbose(args ...interface{}) {
	if verbose {
		log(os.Stdout, "", args...)
	}
}

func logWarning(args ...interface{}) {
	log(os.Stderr, "WARNING:", args...)
}
