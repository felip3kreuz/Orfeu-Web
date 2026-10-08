//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func setUTF8Console() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setOut := kernel32.NewProc("SetConsoleOutputCP")
	setIn := kernel32.NewProc("SetConsoleCP")
	getMode := kernel32.NewProc("GetConsoleMode")
	setMode := kernel32.NewProc("SetConsoleMode")

	_, _, _ = setOut.Call(uintptr(65001))
	_, _, _ = setIn.Call(uintptr(65001))

	h := uintptr(os.Stdout.Fd())
	var mode uint32
	r, _, _ := getMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	if r != 0 {
		const enableVirtualTerminalProcessing = 0x0004
		r2, _, _ := setMode.Call(h, uintptr(mode|enableVirtualTerminalProcessing))
		if r2 != 0 {
			colorCapable = true
			colorEnabled = true
		}
	}
}

func showNativeError(title, message string) {
	if diagConsole == "true" {
		fmt.Fprintln(os.Stderr, title+": "+message)
	}
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(message)
	const mbOK = 0x00000000
	const mbIconError = 0x00000010
	_, _, _ = messageBox.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), mbOK|mbIconError)
}
