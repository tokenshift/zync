package main

import "fmt"

// Looks for the named flag in the list of arguments. If it is found, returns
// true and the argument list minus that argument.
func argFlag(args []string, names ...string) (bool, []string) {
	names = normalizeOptionNames(names...)

	for i, arg := range(args) {
		for _, name := range(names) {
			if arg == name {
				return true, append(args[:i], args[i+1:]...)
			}
		}
	}

	return false, args
}

// Looks for the named option in the argument list. If it is found, returns
// true, the associated (subsequent) value, and the argument list with both of
// those parameters removed. If the option is found with no value, returns true
// and an empty string, and only the option itself removed.
func argOption(args []string, names ...string) (bool, string, []string) {
	names = normalizeOptionNames(names...)

	for i, arg := range(args) {
		for _, name := range(names) {
			if arg == name {
				if len(args) > i + 1 {
					val := args[i+1]
					return true, val, append(args[:i], args[i+2:]...)
				} else {
					return true, "", append(args[:i], args[i+1:]...)
				}
			}
		}
	}

	return false, "", args
}

// Prefixes option names with a single- or double-hyphen, based on whether they
// are single-character or longer, respectively.
func normalizeOptionNames(names ...string) []string {
	for i, name := range(names) {
		if len(name) == 0 {
			continue
		}
		if len(name) == 1 {
			names[i] = fmt.Sprintf("-%s", name)
		} else {
			names[i] = fmt.Sprintf("--%s", name)
		}
	}

	return names
}
