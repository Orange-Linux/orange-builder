package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigFileNames to obsługiwane nazwy pliku konfiguracyjnego projektu -
// zgodnie z opisem można użyć zarówno rozszerzenia .yaml jak i .yml.
var ConfigFileNames = []string{"config.yaml", "config.yml"}

// FindConfigFile szuka pliku config.yaml/config.yml w podanym katalogu.
func FindConfigFile(projectDir string) (string, error) {
	for _, name := range ConfigFileNames {
		p := filepath.Join(projectDir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("nie znaleziono pliku config.yaml ani config.yml w katalogu %s", projectDir)
}

// LoadConfig wczytuje i waliduje konfigurację projektu z podanego katalogu.
func LoadConfig(projectDir string) (*Config, error) {
	path, err := FindConfigFile(projectDir)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("nie można odczytać %s: %w", path, err)
	}
	raw, err := ParseYAML(data)
	if err != nil {
		return nil, fmt.Errorf("błąd parsowania %s: %w", path, err)
	}

	cfg := &Config{}

	distRaw := getMap(raw, "distribution")
	cfg.Distribution = Distribution{
		Name:         getString(distRaw, "name", ""),
		Version:      strings.ToLower(getString(distRaw, "version", "")),
		LeapVersion:  getString(distRaw, "leap_version", ""),
		MicroOSToken: getString(distRaw, "micro_os_token", ""),
		Description:  getString(distRaw, "description", ""),
		License:      getString(distRaw, "license", ""),
	}

	imgRaw := getMap(raw, "image")
	cfg.Image = Image{
		Name:       getString(imgRaw, "name", "orange-linux"),
		Version:    getString(imgRaw, "version", "1.0.0"),
		Filesystem: strings.ToLower(getString(imgRaw, "filesystem", "ext4")),
		Arch:       getString(imgRaw, "arch", "x86_64"),
	}

	deskRaw := getMap(raw, "desktop")
	cfg.Desktop = Desktop{
		Environment: strings.ToLower(getString(deskRaw, "environment", "none")),
	}

	instRaw := getMap(raw, "installer")
	userRaw := getMap(instRaw, "live_user")
	cfg.Installer = Installer{
		Type: strings.ToLower(getString(instRaw, "type", InstallerCalamares)),
		LiveUser: LiveUser{
			Create:    getBool(userRaw, "create", true),
			Username:  getString(userRaw, "username", "live"),
			Password:  getString(userRaw, "password", "live"),
			Autologin: getBool(userRaw, "autologin", true),
		},
	}

	reposRaw := getList(raw, "repositories")
	for _, r := range reposRaw {
		m, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		cfg.Repositories = append(cfg.Repositories, Repository{
			Name: getString(m, "name", ""),
			URL:  getString(m, "url", ""),
		})
	}

	profilesRaw := getList(raw, "profiles")
	for _, p := range profilesRaw {
		m, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		cfg.Profiles = append(cfg.Profiles, Profile{
			Name:    getString(m, "name", ""),
			Type:    strings.ToLower(getString(m, "type", ProfileTypeISO)),
			Default: getBool(m, "default", false),
		})
	}

	signRaw := getMap(raw, "signing")
	cfg.Signing = Signing{
		GPGKeyID: getString(signRaw, "gpg_key_id", ""),
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	// Jeśli zdefiniowano profile, ale żaden nie jest oznaczony jako domyślny,
	// pierwszy z listy staje się domyślny (żeby `ob build` bez --profile
	// zawsze miał jednoznaczny wybór).
	if len(cfg.Profiles) > 0 {
		hasDefault := false
		for _, p := range cfg.Profiles {
			if p.Default {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			cfg.Profiles[0].Default = true
		}
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Distribution.Name == "" {
		return fmt.Errorf("distribution.name jest wymagane w config.yaml")
	}
	switch cfg.Distribution.Version {
	case VersionTumbleweed:
		// brak dodatkowych wymagań
	case VersionLeap:
		if cfg.Distribution.LeapVersion == "" {
			return fmt.Errorf("distribution.leap_version jest wymagane gdy distribution.version: Leap")
		}
	case VersionMicroOS:
		if cfg.Distribution.MicroOSToken == "" {
			return fmt.Errorf("distribution.micro_os_token jest wymagany gdy distribution.version: MicroOS (podaj token rejestracyjny)")
		}
	default:
		return fmt.Errorf("nieznana wartość distribution.version: %q (dozwolone: Tumbleweed, Leap, MicroOS)", cfg.Distribution.Version)
	}
	if cfg.Image.Name == "" {
		return fmt.Errorf("image.name jest wymagane w config.yaml")
	}
	switch cfg.Installer.Type {
	case InstallerCalamares, InstallerNone, "":
	default:
		// dopuszczamy inne nazwy instalatorów (np. własny), tylko ostrzeżenie
		// nie jest tutaj traktowane jako błąd - patrz README.
	}
	if len(cfg.Repositories) == 0 {
		return fmt.Errorf("sekcja repositories jest pusta - podaj co najmniej jedno repozytorium pakietów")
	}

	if len(cfg.Profiles) > 0 {
		seenNames := map[string]bool{}
		defaultCount := 0
		for _, p := range cfg.Profiles {
			if p.Name == "" {
				return fmt.Errorf("profiles: każdy profil musi mieć pole name")
			}
			if seenNames[p.Name] {
				return fmt.Errorf("profiles: zduplikowana nazwa profilu %q", p.Name)
			}
			seenNames[p.Name] = true
			if p.Type != ProfileTypeISO && p.Type != ProfileTypeOEM {
				return fmt.Errorf("profiles: nieznany type %q w profilu %q (dozwolone: iso, oem)", p.Type, p.Name)
			}
			if p.Default {
				defaultCount++
			}
		}
		if defaultCount > 1 {
			return fmt.Errorf("profiles: więcej niż jeden profil oznaczony jako default: true")
		}
	}

	return nil
}

// ProfileNames zwraca listę nazw profili - przydatne do komunikatów błędów.
func ProfileNames(profiles []Profile) []string {
	names := make([]string, 0, len(profiles))
	for _, p := range profiles {
		names = append(names, p.Name)
	}
	return names
}

// --- Poniżej pomocnicze funkcje do bezpiecznego odczytu z map[string]interface{} ---

func getMap(raw map[string]interface{}, key string) map[string]interface{} {
	if raw == nil {
		return map[string]interface{}{}
	}
	if v, ok := raw[key]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	return map[string]interface{}{}
}

func getList(raw map[string]interface{}, key string) []interface{} {
	if raw == nil {
		return nil
	}
	if v, ok := raw[key]; ok {
		if l, ok := v.([]interface{}); ok {
			return l
		}
	}
	return nil
}

func getString(raw map[string]interface{}, key string, def string) string {
	if raw == nil {
		return def
	}
	if v, ok := raw[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return def
}

func getBool(raw map[string]interface{}, key string, def bool) bool {
	if raw == nil {
		return def
	}
	if v, ok := raw[key]; ok && v != nil {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}
