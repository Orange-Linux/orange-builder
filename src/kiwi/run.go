package kiwi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orangebuilder/src/util"
)

// RequiredHostTool to nazwa polecenia, którego Orange Builder wymaga na
// systemie hosta do faktycznego zbudowania obrazu (kiwi-ng "pod spodem"),
// jeśli budowanie odbywa się bezpośrednio na hoście (bez kontenera).
const RequiredHostTool = "kiwi-ng"

// CheckToolchain sprawdza czy kiwi-ng jest zainstalowane na maszynie
// budującej i zwraca czytelny (polski) błąd z podpowiedzią jeśli nie.
func CheckToolchain() error {
	if util.CommandExists(RequiredHostTool) {
		return nil
	}
	return fmt.Errorf(
		"nie znaleziono polecenia %q w PATH.\n"+
			"Zainstaluj go poleceniem:\n\n"+
			"    sudo zypper install python3-kiwi\n",
		RequiredHostTool,
	)
}

// buildArgs buduje wspólne argumenty `system build` dla kiwi-ng, opcjonalnie
// dodając --profile <nazwa> gdy projekt korzysta z wielu profili obrazu.
func buildArgs(descriptionDir, targetDir, profile string) []string {
	args := []string{"system", "build", "--description", descriptionDir, "--target-dir", targetDir}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	return args
}

// BuildOnHost uruchamia kiwi-ng system build bezpośrednio na hoście (kiwi-ng
// musi być już zainstalowane - patrz CheckToolchain) i po sukcesie kopiuje
// gotowy obraz ISO do finalImageDir. profile może być pusty (brak profili
// w config.yaml - pojedynczy, domyślny obraz).
func BuildOnHost(appliance *Appliance, buildOutDir, finalImageDir, imageName, imageVersion, profile string) (string, error) {
	if err := CheckToolchain(); err != nil {
		return "", err
	}
	if err := util.EnsureDir(buildOutDir); err != nil {
		return "", err
	}
	if err := util.EnsureDir(finalImageDir); err != nil {
		return "", err
	}

	err := util.RunCommand("", RequiredHostTool, buildArgs(appliance.Dir, buildOutDir, profile)...)
	if err != nil {
		return "", fmt.Errorf("kiwi-ng zwróciło błąd podczas budowania obrazu: %w", err)
	}

	return collectISO(buildOutDir, finalImageDir, imageName, imageVersion, profile)
}

// BuildInContainer buduje obraz wewnątrz izolowanego kontenera podman
// przygotowanego przez EnsureContainer - używane gdy na hoście brakuje
// kiwi-ng, a użytkownik (lub tryb nieinteraktywny/CI) zgodził się, żeby
// Orange Builder sam o to zadbał. applianceRelDir i outRelDir muszą być
// ścieżkami względnymi wobec absProjectDir (bo to on jest zamontowany
// jako /workspace wewnątrz kontenera).
func BuildInContainer(containerName, applianceRelDir, outRelDir, buildOutDirAbs, finalImageDir, imageName, imageVersion, profile string) (string, error) {
	if err := util.EnsureDir(buildOutDirAbs); err != nil {
		return "", err
	}
	if err := util.EnsureDir(finalImageDir); err != nil {
		return "", err
	}

	if err := RunBuildInContainer(containerName, applianceRelDir, outRelDir, profile); err != nil {
		return "", fmt.Errorf("kiwi-ng (w kontenerze podman %q) zwróciło błąd podczas budowania obrazu: %w", containerName, err)
	}

	return collectISO(buildOutDirAbs, finalImageDir, imageName, imageVersion, profile)
}

// collectISO odnajduje wygenerowany przez kiwi-ng plik .iso w katalogu
// wyjściowym i kopiuje go do finalImageDir pod czytelną nazwą opartą na
// image.name, opcjonalnej nazwie profilu oraz image.version z config.yaml.
func collectISO(buildOutDir, finalImageDir, imageName, imageVersion, profile string) (string, error) {
	isoPath, err := findBuiltISO(buildOutDir)
	if err != nil {
		return "", err
	}

	var finalName string
	if profile != "" {
		finalName = fmt.Sprintf("%s-%s-%s.iso", imageName, profile, imageVersion)
	} else {
		finalName = fmt.Sprintf("%s-%s.iso", imageName, imageVersion)
	}
	finalPath := filepath.Join(finalImageDir, finalName)
	if err := util.CopyFile(isoPath, finalPath); err != nil {
		return "", fmt.Errorf("nie udało się skopiować gotowego obrazu do %s: %w", finalPath, err)
	}
	return finalPath, nil
}

// findBuiltISO szuka pliku .iso wygenerowanego przez kiwi-ng w katalogu
// wyjściowym (kiwi-ng samo nazywa plik na podstawie config.xml, więc
// odnajdujemy go po rozszerzeniu zamiast zakładać sztywną nazwę).
func findBuiltISO(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("nie można odczytać katalogu wyjściowego kiwi-ng %s: %w", dir, err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".iso") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("kiwi-ng zakończyło się bez błędu, ale nie znaleziono pliku .iso w %s", dir)
}
