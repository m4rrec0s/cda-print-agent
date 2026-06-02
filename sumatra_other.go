//go:build !windows

package main

import "fmt"

func getSumatraPath() (string, error) {
	return "", fmt.Errorf("SumatraPDF disponivel apenas no Windows")
}
