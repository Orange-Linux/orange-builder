package kiwi

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"orangebuilder/src/config"
	"orangebuilder/src/util"
)

// containerNamePrefix pozwala rozpoznać, że dany kontener podman został
// utworzony przez Orange Buildera (i że wolno go bezpiecznie usunąć przy
// `ob clean`) - to nie jest cudzy kontener użytkownika.
const containerNamePrefix = "ob-kiwi-"

// baseImageLabel to etykieta podmana, w której zapisujemy jakiego obrazu
// bazowego użyto do utworzenia kontenera - dzięki temu ob potrafi wykryć,
// że distribution.version zmieniło się w config.yaml, i odtworzyć
// kontener od nowa zamiast używać nieaktualnego środowiska.
const baseImageLabel = "ob.baseimage"

// CacheDirName to nazwa katalogu w build/ przechowującego trwały cache
// pobranych pakietów (zypp) między kolejnymi budowaniami w kontenerze.
// Katalog NIE jest usuwany przez zwykłe `ob clean` (tylko `ob clean --all`).
const CacheDirName = "cache"

// ContainerName wylicza deterministyczną nazwę kontenera podman dla danego
// projektu (na podstawie ścieżki absolutnej), tak żeby `ob build` i
// `ob clean` zawsze trafiały w ten sam kontener bez potrzeby trzymania
// dodatkowego pliku ze stanem.
func ContainerName(absProjectDir string) string {
	h := sha1.Sum([]byte(absProjectDir))
	return containerNamePrefix + hex.EncodeToString(h[:])[:12]
}

// CacheDir zwraca ścieżkę do katalogu z trwałym cache'em pakietów danego
// projektu: <projekt>/build/cache/zypp.
func CacheDir(absProjectDir string) string {
	return filepath.Join(absProjectDir, "build", CacheDirName, "zypp")
}

// baseImageFor dobiera obraz kontenera najbliższy wybranej w config.yaml
// wersji openSUSE, żeby budowanie odbywało się w środowisku możliwie
// zbliżonym do systemu docelowego.
func baseImageFor(cfg *config.Config) string {
	switch strings.ToLower(cfg.Distribution.Version) {
	case config.VersionLeap:
		if cfg.Distribution.LeapVersion != "" {
			return "registry.opensuse.org/opensuse/leap:" + cfg.Distribution.LeapVersion
		}
		return "registry.opensuse.org/opensuse/leap:latest"
	case config.VersionMicroOS:
		return "registry.opensuse.org/opensuse/microos:latest"
	default: // Tumbleweed lub nieznane - najbezpieczniejszy wybór
		return "registry.opensuse.org/opensuse/tumbleweed:latest"
	}
}

// CheckPodman sprawdza czy podman jest zainstalowany na hoście.
func CheckPodman() error {
	if util.CommandExists("podman") {
		return nil
	}
	return fmt.Errorf(
		"nie znaleziono polecenia \"podman\" w PATH.\n" +
			"Aby Orange Builder mógł sam przygotować izolowane środowisko z kiwi-ng,\n" +
			"zainstaluj Podmana, np.:\n\n" +
			"    sudo zypper install podman\n",
	)
}

func containerExists(name string) bool {
	return exec.Command("podman", "container", "exists", name).Run() == nil
}

func containerRunning(name string) bool {
	out, err := exec.Command("podman", "inspect", "-f", "{{.State.Running}}", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func kiwiInstalledInContainer(name string) bool {
	return exec.Command("podman", "exec", name, "sh", "-c", "command -v kiwi-ng").Run() == nil
}

// containerBaseImageLabel odczytuje etykietę baseImageLabel istniejącego
// kontenera - pusty string jeśli kontener nie istnieje albo etykiety brak
// (np. kontener sprzed wprowadzenia tej funkcji).
func containerBaseImageLabel(name string) string {
	out, err := exec.Command("podman", "inspect", "-f",
		fmt.Sprintf("{{ index .Config.Labels %q }}", baseImageLabel), name).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// EnsureContainer tworzy (jeśli trzeba) i uruchamia izolowany kontener
// podman z kiwi-ng dla danego projektu, montując cały katalog projektu pod
// /workspace oraz trwały cache pakietów pod /var/cache/zypp wewnątrz
// kontenera. Jeśli wersja dystrybucji w config.yaml zmieniła się od
// ostatniego budowania (inny obraz bazowy), stary kontener jest usuwany
// i tworzony od nowa automatycznie. Zwraca nazwę gotowego do użycia
// kontenera.
func EnsureContainer(absProjectDir string, cfg *config.Config) (string, error) {
	if err := CheckPodman(); err != nil {
		return "", err
	}
	name := ContainerName(absProjectDir)
	image := baseImageFor(cfg)

	if containerExists(name) {
		existingImage := containerBaseImageLabel(name)
		if existingImage != "" && existingImage != image {
			util.Warn("Wersja dystrybucji w config.yaml zmieniła się (poprzedni obraz kontenera: %s, nowy: %s).", existingImage, image)
			util.Step("Odtwarzam kontener od nowa z właściwym obrazem bazowym...")
			if err := util.RunCommand("", "podman", "rm", "-f", name); err != nil {
				return "", fmt.Errorf("nie udało się usunąć nieaktualnego kontenera przed odtworzeniem: %w", err)
			}
		}
	}

	if !containerExists(name) {
		cacheDir := CacheDir(absProjectDir)
		if err := util.EnsureDir(cacheDir); err != nil {
			return "", fmt.Errorf("nie udało się przygotować katalogu cache %s: %w", cacheDir, err)
		}

		util.Step("Tworzę izolowany kontener podman %q (obraz bazowy: %s)", name, image)
		if err := util.RunCommand("", "podman", "create",
			"--name", name,
			"--label", baseImageLabel+"="+image,
			"-v", absProjectDir+":/workspace:Z",
			"-v", cacheDir+":/var/cache/zypp:Z",
			image, "sleep", "infinity",
		); err != nil {
			return "", fmt.Errorf("nie udało się utworzyć kontenera podman: %w", err)
		}
	}

	if !containerRunning(name) {
		if err := util.RunCommand("", "podman", "start", name); err != nil {
			return "", fmt.Errorf("nie udało się uruchomić kontenera podman %q: %w", name, err)
		}
	}

	if !kiwiInstalledInContainer(name) {
		util.Step("Instaluję kiwi-ng wewnątrz kontenera %q (jednorazowo, może to potrwać kilka minut)", name)
		if err := util.RunCommand("", "podman", "exec", name,
			"zypper", "--non-interactive", "install", "python3-kiwi",
		); err != nil {
			return "", fmt.Errorf("nie udało się zainstalować kiwi-ng wewnątrz kontenera: %w", err)
		}
	}

	return name, nil
}

// RunBuildInContainer uruchamia `kiwi-ng system build` wewnątrz kontenera,
// operując na ścieżkach względnych do /workspace (czyli względnych do
// katalogu projektu na hoście, bo jest on tam zamontowany 1:1). profile
// może być pusty (brak profili w config.yaml).
func RunBuildInContainer(containerName, applianceRelDir, outRelDir, profile string) error {
	args := append([]string{"exec", "-w", "/workspace", containerName, "kiwi-ng"},
		buildArgs(applianceRelDir, outRelDir, profile)...)
	return util.RunCommand("", "podman", args...)
}

// RemoveContainer usuwa kontener podman powiązany z danym projektem, jeśli
// istnieje (wywoływane przez `ob clean`). Katalog cache NIE jest usuwany
// przez tę funkcję - to celowe, żeby przetrwał między buildami nawet po
// usunięciu kontenera (patrz `ob clean --all` żeby wyczyścić też cache).
// Brak podmana lub brak kontenera nie jest traktowany jako błąd.
func RemoveContainer(absProjectDir string) error {
	if !util.CommandExists("podman") {
		return nil
	}
	name := ContainerName(absProjectDir)
	if !containerExists(name) {
		return nil
	}
	util.Step("Usuwam kontener podman %q utworzony przez Orange Buildera", name)
	return util.RunCommand("", "podman", "rm", "-f", name)
}
