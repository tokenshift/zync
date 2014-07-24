package main

import "fmt"
import "path/filepath"
import "os"

// Recursively navigates the filesystem from the specified root in alphabetical
// order, returning all files/folders found.
func enumerateFiles(root string) (<-chan FileInfo) {
	out := make(chan FileInfo)

	go func() {
		defer func() {
			close(out)
		}()

		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Fprintln(os.Stderr, "WARNING:", err)
				return nil
			} else {
				fi, err := fileInfo(root, path, info)
				if err == nil {
					out <- fi
				}
				return err
			}
		})
	}()

	return out
}

func fileInfo(root string, path string, info os.FileInfo) (fi FileInfo, err error) {
	path, err = filepath.Rel(root, path)
	if err != nil {
		return
	}

	fi.Path = path
	fi.IsDir = info.IsDir()
	fi.Mode = info.Mode()
	fi.ModTime = info.ModTime()
	fi.Size = info.Size()
	return
}
