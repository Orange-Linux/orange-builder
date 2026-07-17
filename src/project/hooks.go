package project

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"orangebuilder/src/util"
)

// Hook opisuje pojedynczy skrypt z katalogu hooks/, wraz z interpreterem
// jaki należy wywołać, aby go uruchomić wewnątrz chroota obrazu.
type Hook struct {
	Path        string // pełna ścieżka do pliku hooka w projekcie
	Filename    string // sama nazwa pliku (bez katalogu)
	Interpreter string // np. "/bin/bash", "/usr/bin/ruby" itd.
	Package     string // pakiet potrzebny w obrazie żeby hook zadziałał (np. "ruby")
}

// rozszerzenie -> (interpreter, pakiet dostarczający interpreter)
var interpreterByExt = map[string][2]string{
	".sh":   {"/bin/sh", ""}, // sh jest zawsze dostępny w bazowym systemie
	".bash": {"/bin/bash", "bash"},
	".rb":   {"/usr/bin/ruby", "ruby"},
	".py":   {"/usr/bin/python3", "python3"},
	".lua":  {"/usr/bin/lua", "lua"},
	".pl":   {"/usr/bin/perl", "perl"},
}

// LoadHooks wczytuje i sortuje (alfabetycznie po nazwie pliku, tak jak
// robi to run-parts / debianowe hooks/live) skrypty z katalogu hooks/.
// Katalog jest opcjonalny - hooki nie są wymagane.
func LoadHooks(projectDir string) ([]Hook, error) {
	dir := filepath.Join(projectDir, "hooks")
	files, err := util.ListFiles(dir)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	var hooks []Hook
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		info, ok := interpreterByExt[ext]
		if !ok {
			return nil, fmt.Errorf("nieobsługiwane rozszerzenie hooka %q (obsługiwane: .sh, .bash, .rb, .py, .lua, .pl)", f)
		}
		hooks = append(hooks, Hook{
			Path:        f,
			Filename:    filepath.Base(f),
			Interpreter: info[0],
			Package:     info[1],
		})
	}
	return hooks, nil
}

// RequiredPackages zwraca listę dodatkowych pakietów, które trzeba
// zainstalować w obrazie, aby wszystkie hooki mogły zostać uruchomione
// (np. jeśli jest hook .rb, obraz musi mieć zainstalowany pakiet "ruby").
func RequiredPackages(hooks []Hook) []string {
	seen := map[string]bool{}
	var result []string
	for _, h := range hooks {
		if h.Package == "" || seen[h.Package] {
			continue
		}
		seen[h.Package] = true
		result = append(result, h.Package)
	}
	return result
}
