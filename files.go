package main

import "io/ioutil"
import "path"

// Recursively navigates the filesystem from the specified root in alphabetical
// order, returning all files/folders found.
func enumerateFiles(root string) (<-chan string) {
	out := make(chan string)

	go func() {
		entries, err := ioutil.ReadDir(root)
		if err != nil {
			panic(err)
		}

		for _, entry := range(entries) {
			path := path.Join(root, entry.Name())
			out <- path

			// Recursively enumerate the subdirectory.
			if entry.IsDir() {
				recurse := enumerateFiles(path)
				for path = range(recurse) {
					out <- path
				}
			}
		}

		close(out)
	}()

	return out
}
