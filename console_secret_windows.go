//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func readSecretConsole(prompt string) (string, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getMode := kernel32.NewProc("GetConsoleMode")
	setMode := kernel32.NewProc("SetConsoleMode")

	h := uintptr(os.Stdin.Fd())
	var mode uint32
	r, _, _ := getMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		// If stdin is redirected, fall back to normal input.
		return readLocalLine(prompt)
	}
	const enableEchoInput = 0x0004
	fmt.Print(prompt)
	_, _, _ = setMode.Call(h, uintptr(mode&^enableEchoInput))
	defer func() {
		_, _, _ = setMode.Call(h, uintptr(mode))
		fmt.Println()
	}()

	buf := make([]byte, 0, 128)
	one := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(one)
		if n > 0 {
			if one[0] == '\n' {
				break
			}
			if one[0] != '\r' {
				buf = append(buf, one[0])
			}
		}
		if err != nil {
			if len(buf) > 0 {
				break
			}
			return "", err
		}
	}
	return string(buf), nil
}
