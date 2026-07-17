package project

import (
	"path/filepath"
	"strings"

	"orangebuilder/src/util"
)

// FindRPMFiles zwraca listę ścieżek do plików .rpm znajdujących się
// w katalogu rpm-files/ danego projektu. Katalog jest opcjonalny.
func FindRPMFiles(projectDir string) ([]string, error) {
	dir := filepath.Join(projectDir, "rpm-files")
	files, err := util.ListFiles(dir)
	if err != nil {
		return nil, err
	}
	var rpms []string
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f), ".rpm") {
			rpms = append(rpms, f)
		}
	}
	return rpms, nil
}
