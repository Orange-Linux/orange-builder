package kiwi

import (
	"fmt"
	"strings"

	"orangebuilder/src/project"
)

// BuildConfigScript generuje treść pliku config.sh, który kiwi-ng
// automatycznie wykrywa i uruchamia (chrootowany do systemu obrazu) na
// końcu etapu "prepare", czyli już po zainstalowaniu wszystkich pakietów
// z config.xml. To tutaj:
//   1. instalujemy dodatkowe pliki .rpm z rpm-files/ (jeśli są),
//   2. uruchamiamy hooki użytkownika z hooks/ w odpowiedniej kolejności,
//   3. sprzątamy po sobie tymczasowe pliki.
func BuildConfigScript(hasRPMFiles bool, hooks []project.Hook) string {
	var b strings.Builder

	fmt.Fprintf(&b, "#!/bin/bash\n")
	fmt.Fprintf(&b, "# Plik wygenerowany automatycznie przez Orange Builder (ob).\n")
	fmt.Fprintf(&b, "# Edycja ręczna zostanie nadpisana przy kolejnym `ob build`.\n")
	fmt.Fprintf(&b, "set -euo pipefail\n\n")
	fmt.Fprintf(&b, "echo '[orange-builder] config.sh: start'\n\n")

	if hasRPMFiles {
		fmt.Fprintf(&b, "# --- instalacja dodatkowych pakietów .rpm z rpm-files/ ---\n")
		fmt.Fprintf(&b, "if [ -d /tmp/orange-builder-rpms ]; then\n")
		fmt.Fprintf(&b, "  echo '[orange-builder] instaluję pakiety z rpm-files/'\n")
		fmt.Fprintf(&b, "  rpm -Uvh --force --nodeps /tmp/orange-builder-rpms/*.rpm || rpm -Uvh --force /tmp/orange-builder-rpms/*.rpm\n")
		fmt.Fprintf(&b, "  rm -rf /tmp/orange-builder-rpms\n")
		fmt.Fprintf(&b, "fi\n\n")
	}

	if len(hooks) > 0 {
		fmt.Fprintf(&b, "# --- uruchamianie hooków z hooks/ (posortowane alfabetycznie) ---\n")
		for _, h := range hooks {
			hookPath := "/tmp/orange-builder-hooks/" + h.Filename
			fmt.Fprintf(&b, "echo '[orange-builder] hook: %s'\n", h.Filename)
			fmt.Fprintf(&b, "chmod +x %q\n", hookPath)
			fmt.Fprintf(&b, "%s %q\n", h.Interpreter, hookPath)
		}
		fmt.Fprintf(&b, "rm -rf /tmp/orange-builder-hooks\n\n")
	}

	fmt.Fprintf(&b, "echo '[orange-builder] config.sh: koniec'\n")
	fmt.Fprintf(&b, "exit 0\n")
	return b.String()
}

// BuildFlatpakApplyScript generuje skrypt /usr/lib/orange-builder/apply-flatpaks.sh,
// który NIE jest uruchamiany podczas budowania obrazu live, tylko jest
// przeznaczony do wywołania przez instalator (np. jako job Calamares
// "shellprocess") już na zainstalowanym systemie docelowym - to tam, a nie
// w obrazie live, mają się znaleźć aplikacje z packages/install-flatpak.
func BuildFlatpakApplyScript() string {
	var b strings.Builder
	fmt.Fprintf(&b, "#!/bin/bash\n")
	fmt.Fprintf(&b, "# Wygenerowane przez Orange Builder (ob).\n")
	fmt.Fprintf(&b, "# Skrypt do wywołania PO instalacji systemu (np. z Calamares jako\n")
	fmt.Fprintf(&b, "# job typu 'shellprocess', uruchomiony w chroot zainstalowanego systemu).\n")
	fmt.Fprintf(&b, "# Instaluje/usuwa pakiety flatpak wymienione w packages/install-flatpak\n")
	fmt.Fprintf(&b, "# oraz packages/remove-flatpak z projektu Orange Buildera.\n")
	fmt.Fprintf(&b, "set -uo pipefail\n\n")
	fmt.Fprintf(&b, "LIST_DIR=/etc/orange-builder\n\n")
	fmt.Fprintf(&b, "if [ -s \"$LIST_DIR/flatpak-install.list\" ]; then\n")
	fmt.Fprintf(&b, "  while IFS= read -r ref; do\n")
	fmt.Fprintf(&b, "    [ -z \"$ref\" ] && continue\n")
	fmt.Fprintf(&b, "    flatpak install -y --system flathub \"$ref\" || echo \"[orange-builder] nie udało się zainstalować flatpak: $ref\"\n")
	fmt.Fprintf(&b, "  done < \"$LIST_DIR/flatpak-install.list\"\n")
	fmt.Fprintf(&b, "fi\n\n")
	fmt.Fprintf(&b, "if [ -s \"$LIST_DIR/flatpak-remove.list\" ]; then\n")
	fmt.Fprintf(&b, "  while IFS= read -r ref; do\n")
	fmt.Fprintf(&b, "    [ -z \"$ref\" ] && continue\n")
	fmt.Fprintf(&b, "    flatpak uninstall -y --system \"$ref\" || true\n")
	fmt.Fprintf(&b, "  done < \"$LIST_DIR/flatpak-remove.list\"\n")
	fmt.Fprintf(&b, "fi\n\n")
	fmt.Fprintf(&b, "exit 0\n")
	return b.String()
}
