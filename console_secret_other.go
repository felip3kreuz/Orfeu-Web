//go:build !windows

package main

import "fmt"

func readSecretConsole(prompt string) (string, error) {
	fmt.Print("(a entrada pode ficar visível neste sistema) ")
	return readLocalLine(prompt)
}
