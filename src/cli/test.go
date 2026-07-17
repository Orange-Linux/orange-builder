package cli

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"orangebuilder/src/config"
	"orangebuilder/src/kiwi"
	"orangebuilder/src/util"
)

type testOptions struct {
	projectDir       string
	profile          string // --profile NAZWA: który obraz przetestować, gdy jest ich kilka
	forceContainer   bool   // --container
	forceNoContainer bool   // --no-container
	vncPort          int    // --vnc-port PORT (domyślnie 5900, tylko tryb kontenerowy)
}

func parseTestArgs(args []string) (testOptions, error) {
	opts := testOptions{projectDir: ".", vncPort: 5900}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--container":
			opts.forceContainer = true
		case "--no-container":
			opts.forceNoContainer = true
		case "--profile":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("flaga --profile wymaga wartości (nazwy profilu)")
			}
			opts.profile = args[i]
		case "--vnc-port":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("flaga --vnc-port wymaga wartości (numeru portu)")
			}
			port, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("nieprawidłowy numer portu %q", args[i])
			}
			opts.vncPort = port
		default:
			opts.projectDir = a
		}
	}
	return opts, nil
}

func runTest(args []string) error {
	opts, err := parseTestArgs(args)
	if err != nil {
		return err
	}

	absProjectDir, err := filepath.Abs(opts.projectDir)
	if err != nil {
		return err
	}

	cfg, err := config.LoadConfig(absProjectDir)
	if err != nil {
		return err
	}

	buildDir := filepath.Join(absProjectDir, "build")
	isoPath, isoRel, err := findISOToTest(buildDir, absProjectDir, opts.profile)
	if err != nil {
		return err
	}
	util.Info("Obraz do przetestowania: %s", isoPath)

	useContainer, err := decideQEMUContainerUsage(opts, cfg.Image.Arch)
	if err != nil {
		return err
	}

	if useContainer {
		util.Step("Uruchamiam obraz w qemu wewnątrz kontenera podman")
		containerName, err := kiwi.EnsureTestContainer(absProjectDir, cfg, opts.vncPort)
		if err != nil {
			return err
		}
		util.Info("Połącz się klientem VNC pod adresem 127.0.0.1:%d (bez hasła), żeby zobaczyć ekran.", opts.vncPort)
		util.Info("Zamknij klienta VNC i wciśnij Ctrl+C tutaj, żeby zakończyć.")
		return kiwi.RunQEMUInContainer(containerName, isoRel, cfg.Image.Arch)
	}

	util.Step("Uruchamiam obraz w qemu na hoście")
	return kiwi.RunQEMUOnHost(isoPath, cfg.Image.Arch)
}

// findISOToTest odnajduje obraz .iso do przetestowania w build/. Jeśli jest
// ich kilka (np. po `ob build --all-profiles`), wymaga wskazania profilu
// flagą --profile (albo błędu z listą dostępnych obrazów).
func findISOToTest(buildDir, absProjectDir, profile string) (absPath string, relPath string, err error) {
	matches, err := filepath.Glob(filepath.Join(buildDir, "*.iso"))
	if err != nil {
		return "", "", err
	}
	if len(matches) == 0 {
		return "", "", fmt.Errorf("nie znaleziono żadnego obrazu .iso w %s - uruchom najpierw `ob build`", buildDir)
	}

	var chosen string
	switch {
	case len(matches) == 1:
		chosen = matches[0]
	case profile != "":
		for _, m := range matches {
			if strings.Contains(filepath.Base(m), "-"+profile+"-") {
				chosen = m
				break
			}
		}
		if chosen == "" {
			return "", "", fmt.Errorf("nie znaleziono obrazu .iso dla profilu %q wśród:\n  %s", profile, strings.Join(matches, "\n  "))
		}
	default:
		return "", "", fmt.Errorf("znaleziono więcej niż jeden obraz .iso w %s - wskaż który przetestować flagą --profile:\n  %s", buildDir, strings.Join(matches, "\n  "))
	}

	rel, err := filepath.Rel(absProjectDir, chosen)
	if err != nil {
		return "", "", err
	}
	return chosen, rel, nil
}

// decideQEMUContainerUsage odpowiada za tę samą logikę co decideContainerUsage
// (patrz build.go), tylko dla qemu zamiast kiwi-ng.
func decideQEMUContainerUsage(opts testOptions, arch string) (bool, error) {
	if opts.forceNoContainer {
		if err := kiwi.CheckQEMU(arch); err != nil {
			return false, err
		}
		return false, nil
	}
	if opts.forceContainer {
		return true, nil
	}

	hostErr := kiwi.CheckQEMU(arch)
	if hostErr == nil {
		return false, nil
	}

	util.Warn("Nie znaleziono qemu na tym systemie.")
	if !util.IsInteractiveStdin() {
		util.Info("Tryb nieinteraktywny (np. CI) - automatycznie używam izolowanego kontenera podman.")
		return true, nil
	}

	wantsContainer := util.Confirm(
		"Czy Orange Builder ma sam zainstalować qemu w izolowanym kontenerze podman i w nim uruchomić obraz?",
		true,
	)
	if !wantsContainer {
		return false, hostErr
	}
	return true, nil
}
