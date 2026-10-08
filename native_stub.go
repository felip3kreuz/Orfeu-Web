//go:build !windows

package main

import "errors"

func runNativeUI() error { return errors.New("GUI nativa disponível somente no Windows") }
