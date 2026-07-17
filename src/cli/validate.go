package cli

import (
	"path/filepath"

	"orangebuilder/src/config"
	"orangebuilder/src/project"
	"orangebuilder/src/util"
)

func runValidate(args []string) error {
	projectDir := projectPathArg(args)
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	cfg, err := config.LoadConfig(absProjectDir)
	if err != nil {
		return err
	}

	pkgs, err := project.LoadPackages(absProjectDir)
	if err != nil {
		return err
	}
	rpmFiles, err := project.FindRPMFiles(absProjectDir)
	if err != nil {
		return err
	}
	hooks, err := project.LoadHooks(absProjectDir)
	if err != nil {
		return err
	}
	branding := project.LoadBranding(absProjectDir)

	util.Info("config.yaml jest poprawny.")
	util.Info("Dystrybucja: %s (%s)", cfg.Distribution.Name, cfg.Distribution.Version)
	util.Info("Obraz: %s v%s", cfg.Image.Name, cfg.Image.Version)
	util.Info("Środowisko graficzne: %s", cfg.Desktop.Environment)
	util.Info("Instalator: %s", cfg.Installer.Type)
	util.Info("Pakiety do instalacji: %d", len(pkgs.Install))
	util.Info("Pakiety do usunięcia: %d", len(pkgs.Remove))
	util.Info("Aplikacje flatpak (instalator): %d do instalacji, %d do usunięcia",
		len(pkgs.InstallFlatpak), len(pkgs.RemoveFlatpak))
	util.Info("Pliki .rpm: %d", len(rpmFiles))
	util.Info("Hooki: %d", len(hooks))
	if len(cfg.Profiles) > 0 {
		util.Info("Profile obrazu: %s", joinProfileSummary(cfg.Profiles))
	} else {
		util.Info("Profile obrazu: brak (pojedynczy, domyślny obraz)")
	}
	if cfg.Signing.GPGKeyID != "" {
		util.Info("Podpis GPG: skonfigurowany (klucz %s)", cfg.Signing.GPGKeyID)
	} else {
		util.Info("Podpis GPG: brak (obraz będzie miał tylko sumę SHA256)")
	}

	warnings := 0

	if len(pkgs.Install) == 0 {
		util.Warn("packages/install jest puste - obraz będzie zawierał tylko pakiety bazowe/desktop, bez żadnych dodatkowych aplikacji")
		warnings++
	}
	if (len(pkgs.InstallFlatpak) > 0 || len(pkgs.RemoveFlatpak) > 0) && cfg.Installer.Type != config.InstallerCalamares {
		util.Warn("masz listę flatpak (install-flatpak/remove-flatpak), ale installer.type nie jest ustawiony na \"calamares\" - flatpaki nigdzie się nie zainstalują")
		warnings++
	}
	if cfg.Installer.Type == config.InstallerCalamares && !branding.Any() {
		util.Warn("brak plików w brandings/ (logo.png / wallpaper.png / banner.png) - instalator Calamares wystartuje bez własnego brandingu")
		warnings++
	}
	if warnings == 0 {
		util.Info("Brak ostrzeżeń.")
	}
	return nil
}

func joinProfileSummary(profiles []config.Profile) string {
	s := ""
	for i, p := range profiles {
		if i > 0 {
			s += ", "
		}
		s += p.Name + " (" + p.Type + ")"
		if p.Default {
			s += " [domyślny]"
		}
	}
	return s
}
