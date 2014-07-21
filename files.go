package main

import "fmt"
import "path/filepath"
import "os"

// Recursively navigates the filesystem from the specified root in alphabetical
// order, returning all files/folders found.
func enumerateFiles(root string) (<-chan string) {
	out := make(chan string)

	go func() {
	  defer func() {
	    close(out)
    }()

	  filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
	    if err != nil {
        fmt.Fprintln(os.Stderr, "WARNING:", err)
        return nil
      } else {
        path, err = filepath.Rel(root, path)
        if err == nil {
          out <- path
        }
        return err
      }
    })
  }()

	return out
}
