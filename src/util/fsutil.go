package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileExists zwraca true jeśli podana ścieżka istnieje (plik lub katalog).
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir zwraca true jeśli ścieżka istnieje i jest katalogiem.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// EnsureDir tworzy katalog (wraz z rodzicami) jeśli nie istnieje.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// ReadListFile czyta plik "lista pakietów" (jedna pozycja na linię).
// Puste linie oraz linie zaczynające się od '#' (komentarze) są pomijane.
// Jeżeli plik nie istnieje, zwracana jest pusta lista bez błędu - część
// plików w strukturze projektu (np. packages/remove) jest opcjonalna.
func ReadListFile(path string) ([]string, error) {
	var result []string
	if !FileExists(path) {
		return result, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("nie można otworzyć pliku %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		result = append(result, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("błąd odczytu pliku %s: %w", path, err)
	}
	return result, nil
}

// CopyFile kopiuje pojedynczy plik zachowując uprawnienia.
func CopyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// CopyDir rekurencyjnie kopiuje całą zawartość katalogu src do dst,
// zachowując strukturę katalogów oraz uprawnienia plików (przydatne dla
// katalogu files/, który jest nakładką kopiowaną do obrazu ISO).
// Jeśli src nie istnieje, funkcja nic nie robi (katalog jest opcjonalny).
func CopyDir(src, dst string) error {
	if !FileExists(src) {
		return nil
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return EnsureDir(target)
		}
		// Obsługa dowiązań symbolicznych - odtwarzamy je zamiast kopiować cel.
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			_ = os.Remove(target)
			return os.Symlink(linkTarget, target)
		}
		return CopyFile(path, target)
	})
}

// ListFiles zwraca listę plików (nie katalogów) bezpośrednio wewnątrz
// podanego katalogu, posortowaną alfabetycznie. Jeśli katalog nie istnieje,
// zwracana jest pusta lista.
func ListFiles(dir string) ([]string, error) {
	var result []string
	if !FileExists(dir) {
		return result, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			result = append(result, filepath.Join(dir, e.Name()))
		}
	}
	return result, nil
}

// RunCommand uruchamia polecenie systemowe przekazując stdout/stderr
// bezpośrednio do terminala użytkownika (przydatne dla kiwi-ng, którego
// output chcemy widzieć na żywo podczas budowania obrazu).
func RunCommand(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// CommandExists sprawdza czy dane polecenie jest dostępne w PATH.
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
