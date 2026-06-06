//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const mutexName = "CestoDAmore_PrintAgent_SingleInstance"

var (
	modkernel32     = windows.NewLazyDLL("kernel32.dll")
	procCreateMutex = modkernel32.NewProc("CreateMutexW")
)

const errorAlreadyExists = 183

func AcquireSingleInstanceLock() (bool, uintptr) {
	name, _ := syscall.UTF16PtrFromString(mutexName)
	handle, _, err := procCreateMutex.Call(
		0,
		1, // bInitialOwner = true
		uintptr(unsafe.Pointer(name)),
	)

	if handle == 0 {
		return false, 0
	}

	if err == syscall.Errno(errorAlreadyExists) {
		windows.CloseHandle(windows.Handle(handle))
		return false, 0
	}

	return true, handle
}

func ReleaseSingleInstanceLock(handle uintptr) {
	if handle != 0 {
		windows.CloseHandle(windows.Handle(handle))
	}
}

func FocusExistingInstance() {
	moduser32 := windows.NewLazyDLL("user32.dll")
	findWindow := moduser32.NewProc("FindWindowW")
	showWindow := moduser32.NewProc("ShowWindow")
	setForeground := moduser32.NewProc("SetForegroundWindow")

	title, _ := syscall.UTF16PtrFromString("CdA: Print Agent")
	hwnd, _, _ := findWindow.Call(0, uintptr(unsafe.Pointer(title)))

	if hwnd != 0 {
		const swRestore = uintptr(9)
		showWindow.Call(hwnd, swRestore)
		setForeground.Call(hwnd)
	}
}
