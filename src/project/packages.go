package project

import (
	"path/filepath"

	"orangebuilder/src/util"
)

// Packages trzyma odczytane listy pakietów z katalogu packages/ projektu.
type Packages struct {
	Install         []string // packages/install
	Remove          []string // packages/remove (opcjonalny)
	InstallFlatpak  []string // packages/install-flatpak (opcjonalny)
	RemoveFlatpak   []string // packages/remove-flatpak (opcjonalny)
}

// LoadPackages wczytuje wszystkie listy pakietów z katalogu projektu.
// Jedynie packages/install jest wymagany - pozostałe pliki są opcjonalne
// i mogą nie istnieć lub być puste.
func LoadPackages(projectDir string) (*Packages, error) {
	dir := filepath.Join(projectDir, "packages")

	install, err := util.ReadListFile(filepath.Join(dir, "install"))
	if err != nil {
		return nil, err
	}
	remove, err := util.ReadListFile(filepath.Join(dir, "remove"))
	if err != nil {
		return nil, err
	}
	installFlatpak, err := util.ReadListFile(filepath.Join(dir, "install-flatpak"))
	if err != nil {
		return nil, err
	}
	removeFlatpak, err := util.ReadListFile(filepath.Join(dir, "remove-flatpak"))
	if err != nil {
		return nil, err
	}

	return &Packages{
		Install:        install,
		Remove:         remove,
		InstallFlatpak: installFlatpak,
		RemoveFlatpak:  removeFlatpak,
	}, nil
}
