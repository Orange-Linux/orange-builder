package kiwi

import (
	"fmt"
	"os/exec"
	"strings"

	"orangebuilder/src/config"
	"orangebuilder/src/util"
)

// testContainerPrefix pozwala rozpoznać kontener podman utworzony przez
// `ob test` (osobny od kontenera budującego, bo wymaga opublikowanego
// portu VNC ustawionego już w momencie `podman create`).
const testContainerPrefix = "ob-test-"

// QEMUBinaryFor zwraca oczekiwaną nazwę binarki qemu dla danej architektury
// obrazu (image.arch w config.yaml).
func QEMUBinaryFor(arch string) string {
	switch strings.ToLower(arch) {
	case "aarch64", "arm64":
		return "qemu-system-aarch64"
	default:
		return "qemu-system-x86_64"
	}
}

// ovmfCandidates to typowe ścieżki firmware UEFI (OVMF) na różnych
// dystrybucjach hosta - używane tylko best-effort, jeśli się znajdą.
// Nasze obrazy mają firmware="efi" w config.xml, więc uruchomienie z OVMF
// daje wierniejszy test niż czyste legacy BIOS, ale nie jest wymagane -
// obrazy hybrydowe (hybrid="true") zwykle wciążą wystartują też bez niego.
var ovmfCandidates = []string{
	"/usr/share/qemu/ovmf-x86_64-code.bin", // openSUSE
	"/usr/share/OVMF/OVMF_CODE.fd",         // Debian/Ubuntu
	"/usr/share/edk2/ovmf/OVMF_CODE.fd",    // Fedora
}

func findOVMF() string {
	for _, p := range ovmfCandidates {
		if util.FileExists(p) {
			return p
		}
	}
	return ""
}

// CheckQEMU sprawdza czy binarka qemu dla danej architektury jest dostępna
// na hoście.
func CheckQEMU(arch string) error {
	bin := QEMUBinaryFor(arch)
	if util.CommandExists(bin) {
		return nil
	}
	return fmt.Errorf(
		"nie znaleziono polecenia %q w PATH.\n"+
			"Zainstaluj qemu, np.:\n\n"+
			"    sudo zypper install qemu-x86 qemu-tools\n",
		bin,
	)
}

// RunQEMUOnHost uruchamia obraz ISO bezpośrednio na hoście, w zwykłym oknie
// qemu (SDL/GTK, zależnie od tego z czym qemu zostało skompilowane).
func RunQEMUOnHost(isoPath, arch string) error {
	if err := CheckQEMU(arch); err != nil {
		return err
	}
	bin := QEMUBinaryFor(arch)
	args := []string{"-m", "2048", "-smp", "2", "-cdrom", isoPath, "-boot", "d"}
	if ovmf := findOVMF(); ovmf != "" {
		args = append(args, "-bios", ovmf)
	}
	util.Info("Uruchamiam: %s %s", bin, strings.Join(args, " "))
	util.Info("Zamknij okno QEMU, żeby zakończyć.")
	return util.RunCommand("", bin, args...)
}

// TestContainerName wylicza deterministyczną nazwę kontenera podman
// używanego przez `ob test` dla danego projektu.
func TestContainerName(absProjectDir string) string {
	return testContainerPrefix + ContainerName(absProjectDir)[len(containerNamePrefix):]
}

func qemuInstalledInContainer(name, bin string) bool {
	return exec.Command("podman", "exec", name, "sh", "-c", "command -v "+bin).Run() == nil
}

// EnsureTestContainer tworzy (jeśli trzeba) i uruchamia dedykowany kontener
// podman do testowania obrazów w qemu, z ekranem wystawionym przez VNC na
// porcie hosta 127.0.0.1:<vncPort>. To OSOBNY kontener od tego używanego do
// budowania (ContainerName) - port musi być opublikowany już przy tworzeniu
// kontenera, więc nie da się go dołożyć do istniejącego kontenera build.
func EnsureTestContainer(absProjectDir string, cfg *config.Config, vncPort int) (string, error) {
	if err := CheckPodman(); err != nil {
		return "", err
	}
	name := TestContainerName(absProjectDir)
	image := baseImageFor(cfg)

	if containerExists(name) {
		existingImage := containerBaseImageLabel(name)
		if existingImage != "" && existingImage != image {
			util.Warn("Wersja dystrybucji w config.yaml zmieniła się - odtwarzam kontener testowy od nowa.")
			if err := util.RunCommand("", "podman", "rm", "-f", name); err != nil {
				return "", fmt.Errorf("nie udało się usunąć nieaktualnego kontenera testowego: %w", err)
			}
		}
	}

	if !containerExists(name) {
		util.Step("Tworzę kontener testowy podman %q (ekran VNC: 127.0.0.1:%d)", name, vncPort)
		if err := util.RunCommand("", "podman", "create",
			"--name", name,
			"--label", baseImageLabel+"="+image,
			"-p", fmt.Sprintf("127.0.0.1:%d:5900", vncPort),
			"-v", absProjectDir+":/workspace:Z",
			image, "sleep", "infinity",
		); err != nil {
			return "", fmt.Errorf("nie udało się utworzyć kontenera testowego: %w", err)
		}
	}

	if !containerRunning(name) {
		if err := util.RunCommand("", "podman", "start", name); err != nil {
			return "", fmt.Errorf("nie udało się uruchomić kontenera testowego %q: %w", name, err)
		}
	}

	bin := QEMUBinaryFor(cfg.Image.Arch)
	if !qemuInstalledInContainer(name, bin) {
		util.Step("Instaluję qemu wewnątrz kontenera testowego %q (jednorazowo)", name)
		if err := util.RunCommand("", "podman", "exec", name,
			"zypper", "--non-interactive", "install", "qemu-x86", "qemu-tools",
		); err != nil {
			return "", fmt.Errorf("nie udało się zainstalować qemu wewnątrz kontenera: %w", err)
		}
	}

	return name, nil
}

// RunQEMUInContainer uruchamia obraz ISO w kontenerze testowym. isoRelPath
// musi być ścieżką względną wobec katalogu projektu (bo jest on zamontowany
// jako /workspace wewnątrz kontenera).
func RunQEMUInContainer(containerName, isoRelPath, arch string) error {
	bin := QEMUBinaryFor(arch)
	args := []string{
		"exec", "-w", "/workspace", containerName, bin,
		"-m", "2048", "-smp", "2",
		"-cdrom", isoRelPath,
		"-boot", "d",
		"-vnc", "0.0.0.0:0",
	}
	return util.RunCommand("", "podman", args...)
}

// RemoveTestContainer usuwa kontener testowy podman powiązany z danym
// projektem, jeśli istnieje (wywoływane przez `ob clean`).
func RemoveTestContainer(absProjectDir string) error {
	if !util.CommandExists("podman") {
		return nil
	}
	name := TestContainerName(absProjectDir)
	if !containerExists(name) {
		return nil
	}
	util.Step("Usuwam kontener testowy podman %q", name)
	return util.RunCommand("", "podman", "rm", "-f", name)
}
