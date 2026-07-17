package cli

import "fmt"

func printHelp() {
	fmt.Print(`Orange Builder (ob) - narzędzie do budowania obrazów live ISO
dla dystrybucji opartych o openSUSE (Orange Linux i pochodne).

Użycie:
  ob <komenda> [ścieżka-do-projektu] [flagi]

Dostępne komendy:
  build [ścieżka]      Buduje obraz ISO na podstawie projektu w podanej
                        ścieżce (domyślnie: bieżący katalog). Wynikowy
                        obraz (+ .sha256, opcjonalnie .asc) trafia do
                        <projekt>/build/.
      --profile NAZWA     buduj tylko wskazany profil (patrz profiles: w config.yaml)
      --all-profiles       buduj wszystkie zdefiniowane profile po kolei
      --container          wymuś budowanie w kontenerze podman, bez pytania
      --no-container       nigdy nie proponuj kontenera (błąd gdy brak kiwi-ng)

  test [ścieżka]       Uruchamia zbudowany obraz .iso z build/ w qemu -
                        do szybkiego sprawdzenia czy w ogóle się uruchamia,
                        zanim wypalisz go na pendrive'a.
      --profile NAZWA     który obraz przetestować, gdy jest ich kilka
      --container          wymuś qemu w kontenerze podman (ekran przez VNC)
      --no-container       nigdy nie proponuj kontenera
      --vnc-port PORT      port hosta dla ekranu VNC w trybie kontenerowym (domyślnie 5900)

  clean [ścieżka]      Usuwa katalog build/ (poza build/cache/ - trwałym
                        cache pakietów) oraz kontenery podman utworzone
                        przez ob dla tego projektu.
      --all                usuń też build/cache/ (pełne czyszczenie)

  init [ścieżka]       Tworzy szkielet nowego projektu (config.yaml,
                        packages/, rpm-files/, files/, hooks/, brandings/)
                        w podanej ścieżce (domyślnie: bieżący katalog).

  validate [ścieżka]   Sprawdza poprawność config.yaml i struktury
                        projektu, w tym ostrzega o pustych listach
                        pakietów i innych podejrzanych brakach.

  version               Wypisuje wersję narzędzia.
  help                   Wypisuje tę pomoc.

Przykłady:
  ob build
  ob build ~/projekty/orange-linux --container
  ob build --all-profiles
  ob test --profile live
  ob clean
  ob clean --all
  ob init moj-nowy-system

Struktura projektu Orange Buildera:
  config.yaml               Opis dystrybucji, obrazu, środowiska graficznego,
                             instalatora, repozytoriów, opcjonalnych profili
                             obrazu i opcjonalnego podpisu GPG.
  packages/install            Lista pakietów do zainstalowania (linia po linii).
  packages/remove               Lista pakietów do usunięcia (opcjonalny).
  packages/install-flatpak      Lista aplikacji flatpak dla instalatora (opcjonalny).
  packages/remove-flatpak       Lista aplikacji flatpak do usunięcia (opcjonalny).
  rpm-files/*.rpm                Dodatkowe pakiety .rpm instalowane wprost z pliku.
  files/*                        Pliki kopiowane do obrazu po instalacji pakietów.
  brandings/logo.png              Branding instalatora Calamares (opcjonalny).
  brandings/wallpaper.png          Branding instalatora Calamares (opcjonalny).
  brandings/banner.png              Branding instalatora Calamares (opcjonalny).
  hooks/*                          Skrypty (.sh/.bash/.rb/.py/.lua/.pl) uruchamiane
                                    podczas budowania obrazu, w kolejności alfabetycznej.
  build/cache/zypp                  Trwały cache pobranych pakietów (tryb kontenerowy) -
                                     przetrwa "ob clean", usuwany tylko przez "ob clean --all".
`)
}

func printVersion() {
	fmt.Printf("Orange Builder (ob) v%s\n", Version)
}
