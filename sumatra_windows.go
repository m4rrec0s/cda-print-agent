//go:build windows

package main

import (
	_ "embed"
	"os"
	"path/filepath"
	"sync"
)

//go:embed assets/SumatraPDF.exe
var sumatraBinary []byte

var (
	sumatraExtractedPath string
	sumatraOnce          sync.Once
)

// getSumatraPath extrai o SumatraPDF.exe para o diretório temp na primeira
// chamada e reutiliza o mesmo path nas chamadas seguintes.
func getSumatraPath() (string, error) {
	var extractErr error
	sumatraOnce.Do(func() {
		dir := filepath.Join(os.TempDir(), "cda-print-agent", "bin")
		if err := os.MkdirAll(dir, 0755); err != nil {
			extractErr = err
			return
		}

		path := filepath.Join(dir, "SumatraPDF.exe")

		if _, err := os.Stat(path); err == nil {
			sumatraExtractedPath = path
			return
		}

		if err := os.WriteFile(path, sumatraBinary, 0755); err != nil {
			extractErr = err
		 return
		}

		sumatraExtractedPath = path
	})

	if extractErr != nil {
		return "", extractErr
	}
	return sumatraExtractedPath, nil
}
