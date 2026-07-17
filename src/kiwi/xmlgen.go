package kiwi

import (
	"fmt"
	"strings"

	"orangebuilder/src/config"
	"orangebuilder/src/project"
)

// desktopPatterns mapuje nazwę środowiska graficznego z config.yaml na
// wzorce (patterns) openSUSE, które trzeba zainstalować w obrazie.
var desktopPatterns = map[string][]string{
	"kde":    {"kde_plasma", "kde_plasma_workspace"},
	"plasma": {"kde_plasma", "kde_plasma_workspace"},
	"gnome":  {"gnome", "gnome_basic"},
	"xfce":   {"xfce", "xfce_basic"},
	"none":   {},
	"":       {},
}

// xmlEscape ucieka znaki specjalne XML w prostych wartościach tekstowych.
func xmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}

// BuildConfigXML generuje treść pliku config.xml (opis obrazu KIWI-NG) na
// podstawie sparsowanej konfiguracji projektu, list pakietów, dodatkowych
// pakietów wymaganych przez hooki oraz informacji czy są dołączone pliki
// .rpm (wtedy dodajemy krok instalacji w config.sh, a nie tutaj).
//
// Jeśli cfg.Profiles jest puste, generowany jest pojedynczy, domyślny obraz
// ISO - dokładnie tak jak przed wprowadzeniem wsparcia dla profili (pełna
// kompatybilność wsteczna). Jeśli cfg.Profiles zawiera wpisy, generowany
// jest blok <profiles> oraz osobny <preferences profiles="..."> dla każdego
// z nich (pakiety i repozytoria pozostają WSPÓLNE dla wszystkich profili -
// to świadome uproszczenie, patrz README).
func BuildConfigXML(cfg *config.Config, pkgs *project.Packages, extraPackages []string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
	fmt.Fprintf(&b, "<!-- Plik wygenerowany automatycznie przez Orange Builder (ob). -->\n")
	fmt.Fprintf(&b, "<!-- Edycja ręczna zostanie nadpisana przy kolejnym `ob build`. -->\n")
	fmt.Fprintf(&b, "<image schemaversion=\"7.4\" name=\"%s\">\n", xmlEscape(cfg.Image.Name))

	fmt.Fprintf(&b, "  <description type=\"system\">\n")
	fmt.Fprintf(&b, "    <author>Orange Builder</author>\n")
	fmt.Fprintf(&b, "    <contact>orange-builder@localhost</contact>\n")
	fmt.Fprintf(&b, "    <specification>%s</specification>\n", xmlEscape(cfg.Distribution.Description))
	fmt.Fprintf(&b, "  </description>\n")

	if len(cfg.Profiles) == 0 {
		writePreferences(&b, cfg, "", config.ProfileTypeISO)
	} else {
		writeProfilesDeclaration(&b, cfg.Profiles)
		for _, p := range cfg.Profiles {
			writePreferences(&b, cfg, p.Name, p.Type)
		}
	}

	// Repozytoria pakietów zdefiniowane przez użytkownika w config.yaml.
	// Bez atrybutu "profiles" - obowiązują dla wszystkich profili.
	for _, repo := range cfg.Repositories {
		fmt.Fprintf(&b, "  <repository type=\"rpm-md\" alias=\"%s\">\n", xmlEscape(repo.Name))
		fmt.Fprintf(&b, "    <source path=\"%s\"/>\n", xmlEscape(repo.URL))
		fmt.Fprintf(&b, "  </repository>\n")
	}

	// Jeśli obraz ma automatycznego użytkownika live - definiujemy go tutaj.
	if cfg.Installer.LiveUser.Create {
		fmt.Fprintf(&b, "  <users>\n")
		fmt.Fprintf(&b, "    <user password=\"%s\" home=\"/home/%s\" name=\"%s\" groups=\"users,wheel\"/>\n",
			xmlEscape(cfg.Installer.LiveUser.Password),
			xmlEscape(cfg.Installer.LiveUser.Username),
			xmlEscape(cfg.Installer.LiveUser.Username))
		fmt.Fprintf(&b, "  </users>\n")
	}

	// Pakiety bazowe (bootstrap) - minimalny szkielet systemu.
	fmt.Fprintf(&b, "  <packages type=\"bootstrap\">\n")
	fmt.Fprintf(&b, "    <package name=\"filesystem\"/>\n")
	fmt.Fprintf(&b, "    <package name=\"glibc-locale\"/>\n")
	fmt.Fprintf(&b, "    <package name=\"openSUSE-release\"/>\n")
	fmt.Fprintf(&b, "  </packages>\n")

	// Pakiety właściwego obrazu: desktop + installer + lista użytkownika
	// z packages/install + pakiety wymagane przez hooki (np. ruby, python3).
	fmt.Fprintf(&b, "  <packages type=\"image\">\n")
	fmt.Fprintf(&b, "    <package name=\"kernel-default\"/>\n")
	fmt.Fprintf(&b, "    <package name=\"grub2\"/>\n")
	fmt.Fprintf(&b, "    <package name=\"grub2-x86_64-efi\"/>\n")
	fmt.Fprintf(&b, "    <package name=\"live-add-yast-repos\"/>\n")

	for _, p := range desktopPatterns[strings.ToLower(cfg.Desktop.Environment)] {
		fmt.Fprintf(&b, "    <package name=\"patterns-%s\"/>\n", xmlEscape(p))
	}

	if cfg.Installer.Type == config.InstallerCalamares {
		fmt.Fprintf(&b, "    <package name=\"calamares\"/>\n")
		fmt.Fprintf(&b, "    <package name=\"calamares-branding-openSUSE\"/>\n")
	}

	for _, p := range extraPackages {
		fmt.Fprintf(&b, "    <package name=\"%s\"/>\n", xmlEscape(p))
	}

	for _, p := range pkgs.Install {
		fmt.Fprintf(&b, "    <package name=\"%s\"/>\n", xmlEscape(p))
	}
	fmt.Fprintf(&b, "  </packages>\n")

	// Pakiety do usunięcia - KIWI wspiera osobny typ listy "delete".
	if len(pkgs.Remove) > 0 {
		fmt.Fprintf(&b, "  <packages type=\"delete\">\n")
		for _, p := range pkgs.Remove {
			fmt.Fprintf(&b, "    <package name=\"%s\"/>\n", xmlEscape(p))
		}
		fmt.Fprintf(&b, "  </packages>\n")
	}

	fmt.Fprintf(&b, "</image>\n")
	return b.String()
}

// writeProfilesDeclaration zapisuje blok <profiles> deklarujący nazwy
// wariantów obrazu zdefiniowanych w config.yaml.
func writeProfilesDeclaration(b *strings.Builder, profiles []config.Profile) {
	fmt.Fprintf(b, "  <profiles>\n")
	for _, p := range profiles {
		fmt.Fprintf(b, "    <profile name=\"%s\" description=\"profil %s (%s)\"/>\n",
			xmlEscape(p.Name), xmlEscape(p.Name), xmlEscape(p.Type))
	}
	fmt.Fprintf(b, "  </profiles>\n")
}

// writePreferences zapisuje blok <preferences> dla jednego profilu (albo
// bez atrybutu profiles=, jeśli profileName jest puste - czyli w trybie
// pojedynczego, domyślnego obrazu sprzed wsparcia dla profili).
func writePreferences(b *strings.Builder, cfg *config.Config, profileName, profileType string) {
	attr := ""
	if profileName != "" {
		attr = fmt.Sprintf(" profiles=\"%s\"", xmlEscape(profileName))
	}
	fmt.Fprintf(b, "  <preferences%s>\n", attr)
	fmt.Fprintf(b, "    <version>%s</version>\n", xmlEscape(cfg.Image.Version))
	fmt.Fprintf(b, "    <packagemanager>zypper</packagemanager>\n")
	fmt.Fprintf(b, "    <rpm-excludedocs>true</rpm-excludedocs>\n")
	fmt.Fprintf(b, "    <bootsplash-theme>openSUSE</bootsplash-theme>\n")
	fmt.Fprintf(b, "    <bootloader-theme>openSUSE</bootloader-theme>\n")

	switch profileType {
	case config.ProfileTypeOEM:
		// Obraz instalacyjny/appliance (dysk) zamiast live ISO - minimalne,
		// bezpieczne wartości domyślne do dalszego dostrojenia.
		fmt.Fprintf(b, "    <type image=\"oem\" filesystem=\"%s\">\n", xmlEscape(cfg.Image.Filesystem))
		fmt.Fprintf(b, "      <bootloader name=\"grub2\" timeout=\"10\"/>\n")
		fmt.Fprintf(b, "      <oem-resize>true</oem-resize>\n")
		fmt.Fprintf(b, "    </type>\n")
	default:
		fmt.Fprintf(b, "    <type image=\"iso\" flags=\"overlay\" filesystem=\"%s\" firmware=\"efi\" hybrid=\"true\" hybridpersistent=\"true\">\n",
			xmlEscape(cfg.Image.Filesystem))
		fmt.Fprintf(b, "      <bootloader name=\"grub2\" timeout=\"10\"/>\n")
		if cfg.Installer.Type == config.InstallerCalamares {
			fmt.Fprintf(b, "      <installboot install=\"install\"/>\n")
		}
		fmt.Fprintf(b, "    </type>\n")
	}
	fmt.Fprintf(b, "  </preferences>\n")
}
