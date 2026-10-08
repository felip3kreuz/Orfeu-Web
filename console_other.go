//go:build !windows

package main

import (
	"fmt"
	"os"
)

func setUTF8Console() {
	colorCapable = true
	colorEnabled = true
}

func showNativeError(title, message string) {
	fmt.Fprintln(os.Stderr, title+": "+message)
}
