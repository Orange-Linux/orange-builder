package project

import (
	"path/filepath"

	"orangebuilder/src/util"
)

// FilesOverlayDir zwraca ścieżkę katalogu files/ danego projektu - jest to
// odpowiednik debianowego includes.chroot_after_packages: cała zawartość
// tego katalogu jest kopiowana 1:1 do korzenia systemu obrazu, już PO
// zainstalowaniu pakietów z packages/install.
func FilesOverlayDir(projectDir string) string {
	return filepath.Join(projectDir, "files")
}

// HasFilesOverlay sprawdza czy projekt posiada katalog files/.
func HasFilesOverlay(projectDir string) bool {
	return util.IsDir(FilesOverlayDir(projectDir))
}
