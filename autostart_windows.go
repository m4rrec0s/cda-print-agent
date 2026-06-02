//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const autostartRegValue = "CestoDAmore"

func setAutoStart(enable bool) error {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE,
	)
	if err != nil {
		return err
	}
	defer key.Close()

	if !enable {
		return key.DeleteValue(autostartRegValue)
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	return key.SetStringValue(autostartRegValue,
		fmt.Sprintf(`"%s" --hidden`, filepath.Clean(exe)))
}

func getAutoStartEnabled() bool {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(autostartRegValue)
	return err == nil
}
