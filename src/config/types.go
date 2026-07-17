package config

// Distribution opisuje sekcję "distribution:" z config.yaml.
type Distribution struct {
	Name         string // np. "Orange Linux"
	Version      string // Tumbleweed | Leap | MicroOS
	LeapVersion  string // wymagane tylko gdy Version == Leap, np. "15.6"
	MicroOSToken string // wymagany tylko gdy Version == MicroOS (token rejestracyjny)
	Description  string
	License      string
}

// Image opisuje sekcję "image:" z config.yaml.
type Image struct {
	Name       string // nazwa finalnego pliku ISO (bez rozszerzenia)
	Version    string // wersja obrazu, np. "1.0.0"
	Filesystem string // ext4 | btrfs | xfs...
	Arch       string // x86_64 | aarch64...
}

// Desktop opisuje sekcję "desktop:" z config.yaml.
type Desktop struct {
	Environment string // kde | gnome | xfce | none...
}

// LiveUser opisuje domyślnego użytkownika tworzonego w obrazie live.
type LiveUser struct {
	Create    bool
	Username  string
	Password  string
	Autologin bool
}

// Installer opisuje sekcję "installer:" z config.yaml.
type Installer struct {
	Type     string // calamares | none
	LiveUser LiveUser
}

// Repository opisuje pojedyncze repozytorium pakietów RPM.
type Repository struct {
	Name string
	URL  string
}

// Profile opisuje jeden wariant obrazu (np. "live" ISO oraz "disk" appliance
// do instalacji), definiowany w opcjonalnej sekcji "profiles:" config.yaml.
// Jeśli projekt nie definiuje żadnych profili, Orange Builder buduje
// pojedynczy, domyślny obraz ISO - dokładnie tak jak wcześniej (pełna
// kompatybilność wsteczna).
type Profile struct {
	Name    string
	Type    string // iso (obraz live) | oem (obraz instalacyjny/appliance)
	Default bool
}

// Signing opisuje opcjonalną sekcję "signing:" - sumy kontrolne obrazu są
// liczone zawsze automatycznie; podpis GPG jest tworzony tylko jeśli
// podano gpg_key_id i polecenie `gpg` jest dostępne na hoście.
type Signing struct {
	GPGKeyID string
}

// Config to główna, sparsowana reprezentacja config.yaml.
type Config struct {
	Distribution Distribution
	Image        Image
	Desktop      Desktop
	Installer    Installer
	Repositories []Repository
	Profiles     []Profile
	Signing      Signing
}

const (
	VersionTumbleweed = "tumbleweed"
	VersionLeap       = "leap"
	VersionMicroOS    = "microos"

	InstallerCalamares = "calamares"
	InstallerNone      = "none"

	ProfileTypeISO = "iso"
	ProfileTypeOEM = "oem"
)
