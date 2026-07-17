package cli

import (
	"os"
	"path/filepath"

	"orangebuilder/src/util"
)

const defaultConfigYAML = `# Konfiguracja projektu Orange Builder.
# Pełny opis pól znajdziesz w README.md narzędzia ob.

distribution:
  name: "Orange Linux"
  version: "Tumbleweed"        # Tumbleweed | Leap | MicroOS
  # leap_version: "15.6"       # wymagane tylko gdy version: Leap
  # micro_os_token: ""         # wymagane tylko gdy version: MicroOS
  description: "Orange Linux - przyjazna dystrybucja live oparta o openSUSE"
  license: "GPL-3.0"

image:
  name: "orange-linux"
  version: "1.0.0"
  filesystem: "btrfs"          # ext4 | btrfs | xfs
  arch: "x86_64"

desktop:
  environment: "kde"           # kde | gnome | xfce | none

installer:
  type: "calamares"            # calamares | none
  live_user:
    create: true
    username: "live"
    password: "live"
    autologin: true

repositories:
  - name: "oss"
    url: "http://download.opensuse.org/tumbleweed/repo/oss/"
  - name: "non-oss"
    url: "http://download.opensuse.org/tumbleweed/repo/non-oss/"

# Opcjonalnie: więcej niż jeden wariant obrazu (kiwi "profiles"). Jeśli
# usuniesz/zostawisz pustą tę sekcję, ob zbuduje jeden, domyślny obraz ISO.
# profiles:
#   - name: "live"
#     type: "iso"                # iso (live) | oem (obraz instalacyjny/appliance)
#     default: true
#   - name: "disk"
#     type: "oem"

# Opcjonalnie: podpis GPG obrazu (suma SHA256 jest liczona zawsze,
# niezależnie od tej sekcji).
# signing:
#   gpg_key_id: "TWOJ_ID_KLUCZA_GPG"
`

const samplePackagesInstallComment = `# Jeden pakiet na linię. Linie zaczynające się od '#' są ignorowane.
`

const sampleHookComment = `#!/bin/sh
# Przykładowy hook - hooki są uruchamiane w kolejności alfabetycznej
# nazw plików, wewnątrz obrazu (chroot), po instalacji pakietów.
echo "Hello from Orange Builder hook!"
`

const brandingReadme = `Umieść tutaj (opcjonalnie) pliki brandingu instalatora Calamares:

  logo.png       - zalecane ok. 200x200px, przezroczyste tło
  wallpaper.png  - zalecane min. 1920x1080px
  banner.png     - opcjonalny, zależny od motywu Calamares

Wszystkie trzy są niezależne od siebie - dodaj tylko te, które masz.
Jeśli katalog zostanie pusty, instalator wystartuje z domyślnym,
pustym motywem (zobaczysz o tym ostrzeżenie przy `+"`ob validate`"+`).
`

func runInit(args []string) error {
	projectDir := projectPathArg(args)
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	if err := util.EnsureDir(absProjectDir); err != nil {
		return err
	}

	dirs := []string{"packages", "rpm-files", "files", "hooks", "brandings"}
	for _, d := range dirs {
		if err := util.EnsureDir(filepath.Join(absProjectDir, d)); err != nil {
			return err
		}
	}

	configPath := filepath.Join(absProjectDir, "config.yaml")
	if !util.FileExists(configPath) {
		if err := os.WriteFile(configPath, []byte(defaultConfigYAML), 0o644); err != nil {
			return err
		}
	}

	installPath := filepath.Join(absProjectDir, "packages", "install")
	if !util.FileExists(installPath) {
		if err := os.WriteFile(installPath, []byte(samplePackagesInstallComment), 0o644); err != nil {
			return err
		}
	}

	hookPath := filepath.Join(absProjectDir, "hooks", "00-example.sh")
	if !util.FileExists(hookPath) {
		if err := os.WriteFile(hookPath, []byte(sampleHookComment), 0o755); err != nil {
			return err
		}
	}

	brandingReadmePath := filepath.Join(absProjectDir, "brandings", "README.txt")
	if !util.FileExists(brandingReadmePath) {
		if err := os.WriteFile(brandingReadmePath, []byte(brandingReadme), 0o644); err != nil {
			return err
		}
	}

	util.Step("Utworzono nowy projekt Orange Buildera w %s", absProjectDir)
	util.Info("Uzupełnij config.yaml oraz packages/install, a następnie uruchom `ob build`.")
	return nil
}
