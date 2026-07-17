package kiwi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orangebuilder/src/config"
	"orangebuilder/src/project"
	"orangebuilder/src/util"
)

// Appliance zawiera ścieżki potrzebne po wygenerowaniu opisu obrazu, przydatne
// przy dalszym wywołaniu kiwi-ng oraz przy komunikatach dla użytkownika.
type Appliance struct {
	Dir      string // katalog z config.xml + config.sh + root/ (opis kiwi)
	RootDir  string // Appliance.Dir/root - nakładka na system plików obrazu
}

// GenerateAppliance przygotowuje pełny opis obrazu kiwi-ng na podstawie
// struktury projektu Orange Buildera (packages/, rpm-files/, files/, hooks/,
// config.yaml) i zapisuje go w applianceDir.
func GenerateAppliance(projectDir, applianceDir string, cfg *config.Config) (*Appliance, error) {
	rootDir := filepath.Join(applianceDir, "root")

	if err := util.EnsureDir(applianceDir); err != nil {
		return nil, err
	}
	if err := util.EnsureDir(rootDir); err != nil {
		return nil, err
	}

	pkgs, err := project.LoadPackages(projectDir)
	if err != nil {
		return nil, err
	}
	rpmFiles, err := project.FindRPMFiles(projectDir)
	if err != nil {
		return nil, err
	}
	hooks, err := project.LoadHooks(projectDir)
	if err != nil {
		return nil, err
	}

	// 1) Kopiujemy nakładkę files/ do korzenia systemu obrazu.
	if err := util.CopyDir(project.FilesOverlayDir(projectDir), rootDir); err != nil {
		return nil, err
	}

	// 2) Kopiujemy dodatkowe pliki .rpm do tymczasowego katalogu wewnątrz
	//    obrazu - zostaną zainstalowane i posprzątane przez config.sh.
	if len(rpmFiles) > 0 {
		tmpRPMDir := filepath.Join(rootDir, "tmp", "orange-builder-rpms")
		if err := util.EnsureDir(tmpRPMDir); err != nil {
			return nil, err
		}
		for _, f := range rpmFiles {
			if err := util.CopyFile(f, filepath.Join(tmpRPMDir, filepath.Base(f))); err != nil {
				return nil, err
			}
		}
	}

	// 3) Kopiujemy hooki do tymczasowego katalogu wewnątrz obrazu, będą
	//    wywołane po kolei przez config.sh.
	if len(hooks) > 0 {
		tmpHooksDir := filepath.Join(rootDir, "tmp", "orange-builder-hooks")
		if err := util.EnsureDir(tmpHooksDir); err != nil {
			return nil, err
		}
		for _, h := range hooks {
			if err := util.CopyFile(h.Path, filepath.Join(tmpHooksDir, h.Filename)); err != nil {
				return nil, err
			}
		}
	}

	hasFlatpakLists := len(pkgs.InstallFlatpak) > 0 || len(pkgs.RemoveFlatpak) > 0

	// 4) Zapisujemy listy flatpak (do wykorzystania przez instalator na
	//    systemie docelowym) oraz skrypt pomocniczy je stosujący.
	if hasFlatpakLists {
		obDir := filepath.Join(rootDir, "etc", "orange-builder")
		if err := util.EnsureDir(obDir); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(obDir, "flatpak-install.list"),
			[]byte(strings.Join(pkgs.InstallFlatpak, "\n")+"\n"), 0o644); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(obDir, "flatpak-remove.list"),
			[]byte(strings.Join(pkgs.RemoveFlatpak, "\n")+"\n"), 0o644); err != nil {
			return nil, err
		}
		scriptDir := filepath.Join(rootDir, "usr", "lib", "orange-builder")
		if err := util.EnsureDir(scriptDir); err != nil {
			return nil, err
		}
		scriptPath := filepath.Join(scriptDir, "apply-flatpaks.sh")
		if err := os.WriteFile(scriptPath, []byte(BuildFlatpakApplyScript()), 0o755); err != nil {
			return nil, err
		}
	}

	// 4b) Generujemy pełną konfigurację Calamares (branding + settings.conf +
	//     moduły + ewentualny job flatpak) - tylko jeśli installer.type
	//     w config.yaml jest ustawiony na "calamares".
	if cfg.Installer.Type == config.InstallerCalamares {
		if err := GenerateCalamares(rootDir, projectDir, cfg, hasFlatpakLists); err != nil {
			return nil, fmt.Errorf("nie udało się wygenerować konfiguracji Calamares: %w", err)
		}
	}

	// 5) Generujemy config.xml (opis obrazu) i config.sh (hooki + rpm-files).
	extraPackages := project.RequiredPackages(hooks)
	xmlContent := BuildConfigXML(cfg, pkgs, extraPackages)
	if err := os.WriteFile(filepath.Join(applianceDir, "config.xml"), []byte(xmlContent), 0o644); err != nil {
		return nil, err
	}

	scriptContent := BuildConfigScript(len(rpmFiles) > 0, hooks)
	if err := os.WriteFile(filepath.Join(applianceDir, "config.sh"), []byte(scriptContent), 0o755); err != nil {
		return nil, err
	}

	return &Appliance{Dir: applianceDir, RootDir: rootDir}, nil
}
