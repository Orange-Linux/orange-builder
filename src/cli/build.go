package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"orangebuilder/src/config"
	"orangebuilder/src/kiwi"
	"orangebuilder/src/release"
	"orangebuilder/src/util"
)

// buildOptions to sparsowane flagi komendy `ob build`.
type buildOptions struct {
	projectDir       string
	profile          string // --profile NAZWA: buduj tylko wskazany profil
	allProfiles      bool   // --all-profiles: buduj wszystkie zdefiniowane profile
	forceContainer   bool   // --container: zawsze buduj w kontenerze podman
	forceNoContainer bool   // --no-container: nigdy nie proponuj kontenera
}

// parseBuildArgs wyciąga opcjonalną ścieżkę do projektu oraz flagi z
// argumentów podkomendy `ob build`.
func parseBuildArgs(args []string) (buildOptions, error) {
	opts := buildOptions{projectDir: "."}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--container":
			opts.forceContainer = true
		case "--no-container":
			opts.forceNoContainer = true
		case "--all-profiles":
			opts.allProfiles = true
		case "--profile":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("flaga --profile wymaga wartości (nazwy profilu)")
			}
			opts.profile = args[i]
		default:
			opts.projectDir = a
		}
	}
	return opts, nil
}

func runBuild(args []string) error {
	opts, err := parseBuildArgs(args)
	if err != nil {
		return err
	}

	absProjectDir, err := filepath.Abs(opts.projectDir)
	if err != nil {
		return err
	}

	util.Step("Wczytywanie config.yaml z %s", absProjectDir)
	cfg, err := config.LoadConfig(absProjectDir)
	if err != nil {
		return err
	}
	util.Info("Dystrybucja: %s (%s)", cfg.Distribution.Name, cfg.Distribution.Version)
	util.Info("Obraz: %s v%s [%s, %s]", cfg.Image.Name, cfg.Image.Version, cfg.Image.Filesystem, cfg.Image.Arch)

	relApplianceDir := filepath.Join("build", "kiwi-appliance")
	buildDir := filepath.Join(absProjectDir, "build")
	applianceDir := filepath.Join(absProjectDir, relApplianceDir)

	util.Step("Przygotowywanie opisu obrazu (packages/, rpm-files/, files/, hooks/, Calamares)")
	appliance, err := kiwi.GenerateAppliance(absProjectDir, applianceDir, cfg)
	if err != nil {
		return fmt.Errorf("nie udało się przygotować opisu obrazu: %w", err)
	}

	profilesToBuild, err := resolveProfilesToBuild(cfg, opts)
	if err != nil {
		return err
	}

	useContainer, err := decideContainerUsage(opts)
	if err != nil {
		return err
	}

	var containerName string
	if useContainer {
		util.Step("Przygotowuję izolowany kontener podman z kiwi-ng (pakiety cache'owane w build/cache/)")
		containerName, err = kiwi.EnsureContainer(absProjectDir, cfg)
		if err != nil {
			return err
		}
	}

	for _, profile := range profilesToBuild {
		if profile != "" {
			util.Step("Budowanie profilu %q", profile)
		}

		relOutDir := filepath.Join("build", "kiwi-out")
		if profile != "" {
			// Osobny katalog wyjściowy na profil - żeby przy --all-profiles
			// obrazy kolejnych profili się nie nadpisywały/mieszały.
			relOutDir = filepath.Join("build", "kiwi-out", profile)
		}
		kiwiOutDir := filepath.Join(absProjectDir, relOutDir)

		var isoPath string
		if useContainer {
			util.Step("Budowanie obrazu ISO wewnątrz izolowanego kontenera podman (kiwi-ng)")
			isoPath, err = kiwi.BuildInContainer(containerName, relApplianceDir, relOutDir, kiwiOutDir, buildDir, cfg.Image.Name, cfg.Image.Version, profile)
		} else {
			util.Step("Budowanie obrazu ISO za pomocą kiwi-ng zainstalowanego na hoście")
			isoPath, err = kiwi.BuildOnHost(appliance, kiwiOutDir, buildDir, cfg.Image.Name, cfg.Image.Version, profile)
		}
		if err != nil {
			return err
		}

		if err := release.Finalize(isoPath, cfg); err != nil {
			util.Warn("nie udało się dokończyć sumy kontrolnej/podpisu obrazu: %s", err.Error())
		}

		util.Step("Gotowe! Obraz zapisano w: %s", isoPath)
	}

	return nil
}

// resolveProfilesToBuild ustala listę nazw profili do zbudowania:
//   - brak profili w config.yaml -> jeden element "" (bez --profile w kiwi-ng)
//   - --all-profiles              -> wszystkie zdefiniowane profile
//   - --profile NAZWA             -> tylko wskazany (błąd jeśli nie istnieje)
//   - żadna flaga                 -> tylko profil oznaczony jako default
func resolveProfilesToBuild(cfg *config.Config, opts buildOptions) ([]string, error) {
	if len(cfg.Profiles) == 0 {
		return []string{""}, nil
	}
	if opts.allProfiles {
		return config.ProfileNames(cfg.Profiles), nil
	}
	if opts.profile != "" {
		for _, p := range cfg.Profiles {
			if p.Name == opts.profile {
				return []string{opts.profile}, nil
			}
		}
		return nil, fmt.Errorf("nieznany profil %q (dostępne: %s)", opts.profile, strings.Join(config.ProfileNames(cfg.Profiles), ", "))
	}
	for _, p := range cfg.Profiles {
		if p.Default {
			return []string{p.Name}, nil
		}
	}
	return []string{cfg.Profiles[0].Name}, nil
}

// decideContainerUsage ustala, czy budowanie ma się odbyć w kontenerze
// podman, czy bezpośrednio na hoście za pomocą kiwi-ng:
//  1. --no-container  -> nigdy nie używaj kontenera (zwróć błąd hosta jeśli brak kiwi-ng)
//  2. --container     -> zawsze użyj kontenera, bez pytania
//  3. kiwi-ng jest na hoście -> użyj go bezpośrednio
//  4. kiwi-ng nie jest na hoście:
//     - terminal interaktywny -> zapytaj użytkownika
//     - tryb nieinteraktywny (CI) -> automatycznie użyj kontenera, z informacją
func decideContainerUsage(opts buildOptions) (bool, error) {
	if opts.forceNoContainer {
		if err := kiwi.CheckToolchain(); err != nil {
			return false, err
		}
		return false, nil
	}
	if opts.forceContainer {
		return true, nil
	}

	hostErr := kiwi.CheckToolchain()
	if hostErr == nil {
		return false, nil
	}

	util.Warn("Nie znaleziono kiwi-ng na tym systemie.")
	if !util.IsInteractiveStdin() {
		util.Info("Tryb nieinteraktywny (np. CI) - automatycznie używam izolowanego kontenera podman.")
		return true, nil
	}

	wantsContainer := util.Confirm(
		"Czy Orange Builder ma sam przygotować izolowane środowisko (podman + kiwi-ng) i zbudować w nim obraz?",
		true,
	)
	if !wantsContainer {
		return false, hostErr
	}
	return true, nil
}
