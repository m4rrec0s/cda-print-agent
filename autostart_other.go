//go:build !windows

package main

func setAutoStart(_ bool) error {
	return nil
}

func getAutoStartEnabled() bool {
	return false
}
