package project

import (
	"path/filepath"

	"orangebuilder/src/util"
)

// BrandingFiles trzyma ścieżki do znalezionych plików brandingu z katalogu
// brandings/ projektu. Każde pole jest puste jeśli dany plik nie istnieje -
// wszystkie są opcjonalne.
type BrandingFiles struct {
	Logo      string // brandings/logo.png
	Wallpaper string // brandings/wallpaper.png
	Banner    string // brandings/banner.png
}

// Any zwraca true jeśli znaleziono choć jeden plik brandingu.
func (b BrandingFiles) Any() bool {
	return b.Logo != "" || b.Wallpaper != "" || b.Banner != ""
}

// BrandingDir zwraca ścieżkę katalogu brandings/ danego projektu - leży on
// obok packages/, rpm-files/, files/ i hooks/, w korzeniu projektu.
func BrandingDir(projectDir string) string {
	return filepath.Join(projectDir, "brandings")
}

// LoadBranding wyszukuje logo.png, wallpaper.png i banner.png w katalogu
// brandings/ projektu. Katalog oraz każdy z plików są opcjonalne.
func LoadBranding(projectDir string) BrandingFiles {
	dir := BrandingDir(projectDir)

	pick := func(name string) string {
		p := filepath.Join(dir, name)
		if util.FileExists(p) {
			return p
		}
		return ""
	}

	return BrandingFiles{
		Logo:      pick("logo.png"),
		Wallpaper: pick("wallpaper.png"),
		Banner:    pick("banner.png"),
	}
}
