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

// displayManagerByDesktop mapuje środowisko graficzne na domyślny menedżer
// logowania, jaki Calamares powinien włączyć na systemie docelowym.
var displayManagerByDesktop = map[string]string{
	"kde":    "sddm",
	"plasma": "sddm",
	"gnome":  "gdm",
	"xfce":   "lightdm",
	"none":   "sddm",
	"":       "sddm",
}

// BrandingSlug zamienia nazwę dystrybucji na bezpieczny identyfikator
// używany jako componentName brandingu Calamares oraz nazwa katalogu
// /etc/calamares/branding/<slug>/.
func BrandingSlug(distributionName string) string {
	s := strings.ToLower(strings.TrimSpace(distributionName))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	slug := b.String()
	if slug == "" {
		slug = "orange-linux"
	}
	return slug
}

// GenerateCalamares tworzy pełną konfigurację Calamares (branding,
// settings.conf oraz konfiguracje modułów) bezpośrednio w nakładce root/
// obrazu, tak aby po instalacji pakietu `calamares` na obrazie, Calamares
// uruchamiał się już z brandingiem i sekwencją modułów Orange Buildera,
// zamiast domyślnej konfiguracji z calamares-branding-openSUSE.
//
// Obrazy brandingu (logo.png, wallpaper.png, banner.png) są brane z
// katalogu brandings/ w korzeniu projektu (obok packages/, rpm-files/,
// files/, hooks/) - jeśli dany plik tam nie istnieje, branding.desc po
// prostu go nie referencjonuje.
//
// Jeżeli w katalogu files/ projektu użytkownik dostarczył już własny
// /etc/calamares/settings.conf (czyli plik istnieje w rootDir po
// skopiowaniu nakładki files/), generator NIE nadpisuje go - zakładamy,
// że użytkownik świadomie chce pełną kontrolę nad konfiguracją Calamares.
func GenerateCalamares(rootDir, projectDir string, cfg *config.Config, includeFlatpakJob bool) error {
	settingsPath := filepath.Join(rootDir, "etc", "calamares", "settings.conf")
	if util.FileExists(settingsPath) {
		util.Warn("znaleziono własny /etc/calamares/settings.conf w files/ - pomijam automatyczne generowanie konfiguracji Calamares")
		return nil
	}

	slug := BrandingSlug(cfg.Distribution.Name)
	modulesDir := filepath.Join(rootDir, "etc", "calamares", "modules")
	brandingDir := filepath.Join(rootDir, "etc", "calamares", "branding", slug)

	for _, d := range []string{modulesDir, brandingDir} {
		if err := util.EnsureDir(d); err != nil {
			return err
		}
	}

	brandingFiles := project.LoadBranding(projectDir)
	if brandingFiles.Logo != "" {
		if err := util.CopyFile(brandingFiles.Logo, filepath.Join(brandingDir, "logo.png")); err != nil {
			return err
		}
	}
	if brandingFiles.Wallpaper != "" {
		if err := util.CopyFile(brandingFiles.Wallpaper, filepath.Join(brandingDir, "wallpaper.png")); err != nil {
			return err
		}
	}
	if brandingFiles.Banner != "" {
		if err := util.CopyFile(brandingFiles.Banner, filepath.Join(brandingDir, "banner.png")); err != nil {
			return err
		}
	}
	if !brandingFiles.Any() {
		util.Warn("brak plików w brandings/ (logo.png / wallpaper.png / banner.png) - instalator Calamares wystartuje bez własnego brandingu")
	}

	writers := map[string]string{
		filepath.Join(rootDir, "etc", "calamares", "settings.conf"): buildSettingsConf(slug, includeFlatpakJob),
		filepath.Join(modulesDir, "welcome.conf"):                    buildWelcomeConf(),
		filepath.Join(modulesDir, "locale.conf"):                     buildLocaleConf(),
		filepath.Join(modulesDir, "keyboard.conf"):                   buildKeyboardConf(),
		filepath.Join(modulesDir, "partition.conf"):                  buildPartitionConf(cfg.Image.Filesystem),
		filepath.Join(modulesDir, "users.conf"):                      buildUsersConf(),
		filepath.Join(modulesDir, "summary.conf"):                    buildSummaryConf(),
		filepath.Join(modulesDir, "unpackfs.conf"):                   buildUnpackfsConf(),
		filepath.Join(modulesDir, "mount.conf"):                      buildMountConf(),
		filepath.Join(modulesDir, "umount.conf"):                     buildUmountConf(),
		filepath.Join(modulesDir, "fstab.conf"):                      buildFstabConf(),
		filepath.Join(modulesDir, "localecfg.conf"):                  buildLocaleCfgConf(),
		filepath.Join(modulesDir, "machineid.conf"):                  buildMachineIDConf(),
		filepath.Join(modulesDir, "networkcfg.conf"):                 buildNetworkCfgConf(),
		filepath.Join(modulesDir, "hwclock.conf"):                    buildHwclockConf(),
		filepath.Join(modulesDir, "services-systemd.conf"):           buildServicesSystemdConf(cfg),
		filepath.Join(modulesDir, "displaymanager.conf"):             buildDisplayManagerConf(cfg.Desktop.Environment),
		filepath.Join(modulesDir, "bootloader.conf"):                 buildBootloaderConf(),
		filepath.Join(modulesDir, "finished.conf"):                   buildFinishedConf(cfg.Distribution.Name),
		filepath.Join(brandingDir, "branding.desc"):                  buildBrandingDesc(cfg, slug, brandingFiles),
	}
	if !brandingFiles.Any() {
		writers[filepath.Join(brandingDir, "README-images.txt")] = buildBrandingImagesReadme()
	}

	if includeFlatpakJob {
		writers[filepath.Join(modulesDir, "shellprocess.conf")] = buildShellProcessConf()
	}

	for path, content := range writers {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("nie udało się zapisać %s: %w", path, err)
		}
	}
	return nil
}

func buildSettingsConf(slug string, includeFlatpakJob bool) string {
	var b strings.Builder
	b.WriteString("# Plik wygenerowany automatycznie przez Orange Builder (ob).\n")
	b.WriteString("# Edycja ręczna zostanie nadpisana przy kolejnym `ob build`.\n")
	b.WriteString("# Aby przejąć pełną kontrolę: umieść własny plik w files/etc/calamares/settings.conf\n")
	b.WriteString("# w projekcie - wtedy ten generator zostanie pominięty.\n")
	b.WriteString("---\n")
	b.WriteString("modules-search: [ local ]\n\n")
	b.WriteString("sequence:\n")
	b.WriteString("  - show:\n")
	b.WriteString("    - welcome\n")
	b.WriteString("    - locale\n")
	b.WriteString("    - keyboard\n")
	b.WriteString("    - partition\n")
	b.WriteString("    - users\n")
	b.WriteString("    - summary\n")
	b.WriteString("  - exec:\n")
	b.WriteString("    - partition\n")
	b.WriteString("    - mount\n")
	b.WriteString("    - unpackfs\n")
	b.WriteString("    - machineid\n")
	b.WriteString("    - fstab\n")
	b.WriteString("    - locale\n")
	b.WriteString("    - keyboard\n")
	b.WriteString("    - localecfg\n")
	b.WriteString("    - users\n")
	b.WriteString("    - displaymanager\n")
	b.WriteString("    - networkcfg\n")
	b.WriteString("    - hwclock\n")
	b.WriteString("    - services-systemd\n")
	b.WriteString("    - bootloader\n")
	if includeFlatpakJob {
		b.WriteString("    - shellprocess\n")
	}
	b.WriteString("    - umount\n")
	b.WriteString("  - show:\n")
	b.WriteString("    - finished\n\n")
	fmt.Fprintf(&b, "branding: %s\n", slug)
	b.WriteString("prompt-install: false\n")
	b.WriteString("dont-chroot: false\n")
	return b.String()
}

func buildWelcomeConf() string {
	return `# Wygenerowane przez Orange Builder.
showSupportUrl:       true
showKnownIssuesUrl:   true
showReleaseNotesUrl:  true
requirements:
    requiredStorage:   8
    requiredRam:       2
    internetCheckUrl:  "https://opensuse.org"
    check:
        - storage
        - ram
        - power
    required:
        - storage
        - ram
`
}

func buildLocaleConf() string {
	return `# Wygenerowane przez Orange Builder.
region: "Europe"
zone: "Warsaw"
`
}

func buildKeyboardConf() string {
	return `# Wygenerowane przez Orange Builder.
keyboardModel: "pc105"
writeEtcDefaultKeyboard: true
`
}

func buildPartitionConf(filesystem string) string {
	if filesystem == "" {
		filesystem = "ext4"
	}
	return fmt.Sprintf(`# Wygenerowane przez Orange Builder.
efiSystemPartition:        "/boot/efi"
userSwapChoices:
    - none
    - small
    - suspend
    - file
defaultFileSystemType:     "%s"
availableFileSystemTypes:  [ "ext4", "btrfs", "xfs" ]
partitionLayout:
    - name:       "efi"
      filesystem: "fat32"
      size:       300MiB
      mountPoint: "/boot/efi"
    - name:       "root"
      filesystem: "%s"
      size:       100%%
      mountPoint: "/"
enableLuksAutomatedPartitioning: true
drawNestedPartitions:      false
alwaysShowPartitionLabels: true
allowManualPartitioning:   true
`, filesystem, filesystem)
}

func buildUsersConf() string {
	return `# Wygenerowane przez Orange Builder.
defaultGroups:
    - users
    - wheel
autologinGroup:    "autologin"
doAutologin:       false
sudoersGroup:      "wheel"
setRootPassword:   true
doReuseUser:       false
passwordRequirements:
    minLength: 4
`
}

func buildSummaryConf() string {
	return `# Wygenerowane przez Orange Builder.
# Moduł domyślny - pokazuje podsumowanie wszystkich poprzednich kroków.
`
}

func buildUnpackfsConf() string {
	return `# Wygenerowane przez Orange Builder.
unpack:
    -   source: "/run/live/medium/LiveOS/squashfs.img"
        sourcefs: "squashfs"
        destination: ""
`
}

func buildMountConf() string {
	return `# Wygenerowane przez Orange Builder.
extraMounts: []
extraMountsEfi: []
`
}

func buildUmountConf() string {
	return `# Wygenerowane przez Orange Builder.
# Moduł domyślny - odmontowuje system plików na koniec instalacji.
`
}

func buildFstabConf() string {
	return `# Wygenerowane przez Orange Builder.
efiMountOptions: "umask=0077"
`
}

func buildLocaleCfgConf() string {
	return `# Wygenerowane przez Orange Builder.
# Moduł domyślny - zapisuje wybrane locale do /etc/locale.conf.
`
}

func buildMachineIDConf() string {
	return `# Wygenerowane przez Orange Builder.
systemd: true
dbus: true
symlink: true
`
}

func buildNetworkCfgConf() string {
	return `# Wygenerowane przez Orange Builder.
# Moduł domyślny - kopiuje konfigurację NetworkManager z live do systemu docelowego.
`
}

func buildHwclockConf() string {
	return `# Wygenerowane przez Orange Builder.
utc: true
`
}

func buildServicesSystemdConf(cfg *config.Config) string {
	dm := displayManagerByDesktop[strings.ToLower(cfg.Desktop.Environment)]
	return fmt.Sprintf(`# Wygenerowane przez Orange Builder.
services:
    - name: "NetworkManager"
      mandatory: true
    - name: "%s"
      mandatory: true
`, dm)
}

func buildDisplayManagerConf(desktopEnv string) string {
	dm := displayManagerByDesktop[strings.ToLower(desktopEnv)]
	return fmt.Sprintf(`# Wygenerowane przez Orange Builder.
displaymanagers:
    - %s
basicSetup: true
`, dm)
}

func buildBootloaderConf() string {
	return `# Wygenerowane przez Orange Builder.
# Ścieżki jądra/initrd dopasowane do pakietu kernel-default na openSUSE.
efiBootLoader:       "grub"
kernel:              "/boot/vmlinuz"
img:                 "/boot/initrd"
timeout:             10
installEFIFallback:  true
`
}

func buildFinishedConf(distributionName string) string {
	return fmt.Sprintf(`# Wygenerowane przez Orange Builder dla dystrybucji: %s.
restartNowEnabled:  true
restartNowChecked:  true
restartNowCommand:  "systemctl reboot"
notifyOnFinished:   true
`, distributionName)
}

func buildShellProcessConf() string {
	return `# Wygenerowane przez Orange Builder.
# Uruchamia skrypt instalujący aplikacje flatpak (packages/install-flatpak
# oraz packages/remove-flatpak z projektu) na systemie docelowym, już po
# jego zainstalowaniu, wewnątrz chroota.
dontChroot: false
timeout: 600
script:
    - command: "/usr/lib/orange-builder/apply-flatpaks.sh"
      timeout: 600
`
}

func buildBrandingDesc(cfg *config.Config, slug string, brandingFiles project.BrandingFiles) string {
	name := cfg.Distribution.Name
	version := cfg.Image.Version

	var images strings.Builder
	images.WriteString("images:\n")
	if brandingFiles.Logo != "" {
		images.WriteString("    productLogo:          \"logo.png\"\n")
		images.WriteString("    productIcon:           \"logo.png\"\n")
	}
	if brandingFiles.Wallpaper != "" {
		images.WriteString("    productWallpaper:      \"wallpaper.png\"\n")
	}
	if brandingFiles.Banner != "" {
		// Nie wszystkie motywy Calamares wykorzystują ten klucz - jeśli
		// zainstalowany Calamares go nie obsługuje, po prostu jest ignorowany.
		images.WriteString("    productBanner:         \"banner.png\"\n")
	}

	return fmt.Sprintf(`# Wygenerowane przez Orange Builder.
# Obrazy brandingu pochodzą z katalogu brandings/ w korzeniu projektu
# (brandings/logo.png, brandings/wallpaper.png, brandings/banner.png).
---
componentName: %s

welcomeStyleCalamares: true
welcomeExpandingLogo:  true

strings:
    productName:          "%s"
    shortProductName:     "%s"
    version:              "%s"
    shortVersion:          "%s"
    versionedName:        "%s %s"
    shortVersionedName:    "%s %s"
    bootloaderEntryName:   "%s"
    productUrl:            "https://example.com"
    supportUrl:            "https://example.com/support"
    knownIssuesUrl:        "https://example.com/issues"
    releaseNotesUrl:       "https://example.com/release-notes"

%s
style:
   sidebarBackground:     "#1a1a1a"
   sidebarText:           "#ffffff"
   sidebarTextSelect:     "#3daee9"
`, slug, name, name, version, version, name, version, name, version, name, images.String())
}

func buildBrandingImagesReadme() string {
	return `Brakujące obrazy brandingu Calamares.

Nie znaleziono żadnego pliku w katalogu brandings/ Twojego projektu
(brandings/logo.png, brandings/wallpaper.png, brandings/banner.png), więc
branding.desc nie odwołuje się do żadnych obrazów - instalator wystartuje
z domyślnym, pustym motywem.

Aby dodać własny branding, umieść w projekcie (obok packages/, rpm-files/,
files/, hooks/) katalog:

  brandings/logo.png       (zalecane ok. 200x200px, przezroczyste tło)
  brandings/wallpaper.png  (zalecane min. 1920x1080px)
  brandings/banner.png     (opcjonalny, zależny od motywu Calamares)

Wszystkie trzy pliki są opcjonalne i niezależne od siebie - dodaj tylko
te, które masz. Trafią do obrazu automatycznie przy kolejnym uruchomieniu
polecenia ob build.
`
}
