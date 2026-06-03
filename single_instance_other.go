//go:build !windows

package main

func AcquireSingleInstanceLock() (bool, uintptr) {
	return true, 0
}

func ReleaseSingleInstanceLock(handle uintptr) {}

func FocusExistingInstance() {}
