# Orange Builder (ob)

Narzędzie do budowania obrazów **live ISO** dla dystrybucji opartych o
openSUSE (docelowo: **Orange Linux**), na zasadzie zbliżonej do
`live-build` z Debiana - proste, deklaratywne pliki konfiguracyjne zamiast
ręcznego pisania całego opisu obrazu.

## Dlaczego nie po prostu KIWI?

KIWI (`kiwi-ng`) jest jedynym w pełni wspieranym silnikiem budowania
obrazów dla openSUSE, więc Orange Builder **z niego korzysta pod spodem** -
ale ukrywa jego szczegółową, XML-ową konfigurację za prostą strukturą
katalogów i plikiem `config.yaml`, podobną do tego, co Debian/Ubuntu
oferują w `live-build` (i do czego istnieją gotowce typu
`calamares-settings-debian`, których openSUSE nie ma). `ob` generuje
poprawny opis `kiwi-ng` (config.xml + config.sh) automatycznie na
podstawie Twojego projektu i wywołuje `kiwi-ng system build` w Twoim imieniu.

Wymagania na maszynie budującej:
- openSUSE (Tumbleweed/Leap) z zainstalowanym `python3-kiwi` (`sudo zypper install python3-kiwi`)
  **LUB** samo Podman (`sudo zypper install podman` / `sudo apt-get install podman`)
  - jeśli `kiwi-ng` nie jest zainstalowane, `ob build` zapyta czy sam ma
    przygotować izolowany kontener podman z kiwi-ng (patrz niżej)
- opcjonalnie qemu (`sudo zypper install qemu-x86 qemu-tools`) do `ob test` -
  tak samo, jeśli go nie ma, `ob test` może sam przygotować kontener
- Go >= 1.21 (tylko do zbudowania samego narzędzia `ob`)

### Tryb kontenerowy (podman) - gdy nie masz kiwi-ng/qemu zainstalowanego lokalnie

Jeśli `ob build` nie znajdzie `kiwi-ng` (albo `ob test` nie znajdzie `qemu`)
na hoście:
- w terminalu interaktywnym - zapyta, czy sam ma przygotować izolowane
  środowisko (wymaga zainstalowanego Podmana),
- w trybie nieinteraktywnym (CI, brak TTY) - automatycznie skorzysta
  z kontenera bez pytania.

Możesz też wymusić zachowanie flagą, bez czekania na pytanie (dotyczy
obu komend - `ob build` i `ob test`):

```sh
ob build --container      # zawsze buduj w kontenerze podman
ob build --no-container   # nigdy nie proponuj kontenera (błąd, jeśli brak kiwi-ng)
```

Co się dzieje w trybie kontenerowym (`ob build`):
1. Tworzony jest kontener podman (nazwa deterministyczna na podstawie
   ścieżki projektu, obraz bazowy dobrany do wersji openSUSE z
   `config.yaml`: Tumbleweed/Leap/MicroOS) z zamontowanym katalogiem
   projektu pod `/workspace` oraz katalogiem `build/cache/zypp` pod
   `/var/cache/zypp` (trwały cache pakietów - patrz niżej).
2. Wewnątrz kontenera jednorazowo instalowane jest `python3-kiwi`.
3. `kiwi-ng system build` uruchamiane jest wewnątrz kontenera - obraz ISO
   ląduje bezpośrednio w `<projekt>/build/` na hoście (dzięki bind mountowi).
4. `ob clean` usuwa kontener (ale NIE cache pakietów - patrz niżej).

**Automatyczne odświeżanie kontenera:** jeśli zmienisz `distribution.version`
w `config.yaml` (np. Tumbleweed → Leap) między buildami, `ob build` samo
wykryje niezgodność obrazu bazowego kontenera i odtworzy go od nowa -
nie trzeba ręcznie robić `ob clean`.

### Trwały cache pakietów (`build/cache/`)

W trybie kontenerowym katalog `<projekt>/build/cache/zypp` jest montowany
jako `/var/cache/zypp` wewnątrz kontenera, więc pobrane pakiety przetrwają
między kolejnymi `ob build` (i między odtworzeniami kontenera). `ob clean`
**zachowuje** ten katalog domyślnie - usuwa go dopiero `ob clean --all`.

To rozwiązanie jest "best-effort": przyspiesza wyraźnie samą instalację
kiwi-ng w kontenerze i operacje zypper uruchamiane bezpośrednio na jego
systemie; to, czy kiwi-ng dla KAŻDEJ wersji trafia dokładnie w ten sam
cache przy budowaniu obrazu (zależnie od tego jak dana wersja kiwi-ng
zarządza cache'em zypper w budowanym korzeniu), może się różnić - jeśli
zauważysz, że `build/cache/zypp` zostaje puste mimo wielu buildów, to
sygnał, że Twoja wersja kiwi-ng tego nie respektuje i temat wymaga
dalszego dostrojenia pod konkretną wersję.

## Budowanie narzędzia

```sh
make            # buduje binarkę ./ob
sudo make install   # instaluje do /usr/local/bin/ob
```

## Komendy

```
ob build [ścieżka] [--profile NAZWA|--all-profiles] [--container|--no-container]
                        buduje obraz(y) ISO z projektu w podanej ścieżce
                        (domyślnie: bieżący katalog). Wynik (+ .sha256,
                        opcjonalnie .asc) trafia do <projekt>/build/.
ob test [ścieżka] [--profile NAZWA] [--container|--no-container] [--vnc-port PORT]
                        uruchamia zbudowany obraz .iso z build/ w qemu -
                        szybki test czy w ogóle się odpala.
ob clean [ścieżka] [--all]
                        usuwa build/ (domyślnie zachowuje build/cache/)
                        oraz kontenery podman utworzone dla tego projektu.
ob init [ścieżka]      tworzy szkielet nowego projektu
ob validate [ścieżka]  sprawdza config.yaml i strukturę projektu, ostrzega
                        o pustych listach pakietów i brakującym brandingu
ob version              wypisuje wersję narzędzia
ob help                  lista komend
```

## Struktura projektu obrazu

```
moj-projekt/
├── config.yaml (lub config.yml)
├── packages/
│   ├── install            # pakiety do zainstalowania, jeden na linię
│   ├── remove              # (opcjonalny) pakiety do usunięcia
│   ├── install-flatpak     # (opcjonalny) aplikacje flatpak dla instalatora
│   └── remove-flatpak      # (opcjonalny) aplikacje flatpak do usunięcia
├── rpm-files/
│   └── *.rpm                # dodatkowe pakiety instalowane wprost z pliku
├── files/
│   └── ...                  # pliki kopiowane do obrazu PO instalacji pakietów
│                             # (odpowiednik includes.chroot_after_packages)
├── brandings/
│   ├── logo.png              # (opcjonalny) branding instalatora Calamares
│   ├── wallpaper.png          # (opcjonalny)
│   └── banner.png              # (opcjonalny)
├── hooks/
│   └── 00-cokolwiek.sh          # skrypty .sh/.bash/.rb/.py/.lua/.pl, uruchamiane
│                                 # w kolejności alfabetycznej wewnątrz obrazu
└── build/                        # tworzone przez `ob build` - wyniki + cache
    ├── <image>-<wersja>.iso
    ├── <image>-<wersja>.iso.sha256
    ├── <image>-<wersja>.iso.asc    # tylko jeśli signing.gpg_key_id ustawiony
    └── cache/zypp                  # trwały cache pakietów (przetrwa `ob clean`)
```

Po `ob build` wynik trafia do `moj-projekt/build/<image.name>-<image.version>.iso`
(albo `<image.name>-<profil>-<image.version>.iso`, jeśli projekt korzysta
z wielu profili - patrz niżej).

### `packages/install-flatpak` i `packages/remove-flatpak`

W przeciwieństwie do `packages/install`, te pakiety **nie trafiają do
obrazu live** - Orange Builder zapisuje je jako listy
(`/etc/orange-builder/flatpak-*.list`) oraz skrypt
`/usr/lib/orange-builder/apply-flatpaks.sh` wewnątrz obrazu, który jest
przeznaczony do wywołania przez instalator (np. jako dodatkowy job
`shellprocess` w Calamares) **na systemie docelowym już po instalacji**,
a nie na samym live ISO.

### Calamares (branding + settings.conf + moduły)

Gdy `installer.type: calamares`, Orange Builder generuje w obrazie **pełną,
gotową konfigurację Calamares** (nie tylko instaluje pakiet):

- `/etc/calamares/settings.conf` - sekwencja modułów (welcome, locale,
  keyboard, partition, users, summary / partition, mount, unpackfs,
  machineid, fstab, locale, keyboard, localecfg, users, displaymanager,
  networkcfg, hwclock, services-systemd, bootloader, [shellprocess],
  umount / finished),
- `/etc/calamares/modules/*.conf` - konfiguracja każdego z powyższych
  modułów z sensownymi wartościami domyślnymi (system plików z
  `image.filesystem`, menedżer logowania dobrany do `desktop.environment`:
  KDE→sddm, GNOME→gdm, XFCE→lightdm, itd.),
- `/etc/calamares/branding/<slug>/branding.desc` - branding oparty na
  `distribution.name`/`image.version` (slug wyliczany automatycznie z
  nazwy dystrybucji),
- jeśli projekt ma `packages/install-flatpak` lub `packages/remove-flatpak`
  - dodatkowy moduł `shellprocess` wpięty w sekwencję, wywołujący
    `/usr/lib/orange-builder/apply-flatpaks.sh` na systemie docelowym.

**Obrazy brandingu** - umieść `logo.png`, `wallpaper.png`, `banner.png`
(każdy opcjonalny i niezależny) w katalogu **`brandings/`** w korzeniu
projektu (obok `packages/`, `rpm-files/`, `files/`, `hooks/`) - Orange
Builder skopiuje je automatycznie do brandingu Calamares. Jeśli katalog
jest pusty, `ob validate` o tym ostrzeże, a `branding.desc` po prostu nie
będzie się do żadnych obrazów odwoływać (instalator wystartuje z pustym
motywem zamiast się wysypać).

**Nadpisywanie własną konfiguracją:** jeśli w `files/etc/calamares/settings.conf`
umieścisz własny plik, generator automatycznie go wykryje i pominie
całą automatyczną generację Calamares - przejmujesz wtedy pełną kontrolę.

### Hooki

Rozpoznawane rozszerzenia i wymagane pakiety (dociągane automatycznie do
obrazu, żeby hook miał czym się wykonać):

| rozszerzenie | interpreter        | pakiet    |
|--------------|---------------------|-----------|
| `.sh`        | `/bin/sh`           | (brak, zawsze dostępny) |
| `.bash`      | `/bin/bash`         | `bash`    |
| `.rb`        | `/usr/bin/ruby`     | `ruby`    |
| `.py`        | `/usr/bin/python3`  | `python3` |
| `.lua`       | `/usr/bin/lua`      | `lua`     |
| `.pl`        | `/usr/bin/perl`     | `perl`    |

### Profile obrazu (`profiles:`)

Domyślnie (bez sekcji `profiles:` w `config.yaml`) `ob build` buduje jeden,
prosty obraz ISO - dokładnie jak wcześniej. Możesz też zdefiniować kilka
wariantów w jednym projekcie:

```yaml
profiles:
  - name: "live"
    type: "iso"       # iso = live ISO (hybrydowy, bootowalny z pendrive'a)
    default: true
  - name: "disk"
    type: "oem"        # oem = obraz instalacyjny/appliance (bez live)
```

- `packages/`, `rpm-files/`, `files/`, `hooks/` i `repositories:` są
  **wspólne dla wszystkich profili** (świadome uproszczenie) - różni się
  tylko sam typ/konfiguracja `<preferences>` w kiwi.
- `ob build` bez flag buduje profil oznaczony `default: true` (albo
  pierwszy z listy, jeśli żaden nie jest oznaczony).
- `ob build --profile disk` buduje konkretny profil.
- `ob build --all-profiles` buduje wszystkie po kolei, każdy do osobnego
  pliku `<image>-<profil>-<wersja>.iso`.

### Sumy kontrolne i podpis GPG (`signing:`)

Po każdym udanym budowaniu Orange Builder liczy sumę SHA256 obrazu i
zapisuje ją jako `<iso>.sha256` obok niego (zawsze, automatycznie).
Opcjonalnie, jeśli w `config.yaml` podasz:

```yaml
signing:
  gpg_key_id: "TWOJ_ID_KLUCZA_GPG"
```

i masz zainstalowane `gpg` na hoście, dodatkowo tworzony jest odłączony
podpis `<iso>.asc` (`gpg --detach-sign --armor`). Brak `gpg` przy
skonfigurowanym `gpg_key_id` skutkuje tylko ostrzeżeniem, nie przerywa
budowania.

### `ob test` - szybkie sprawdzenie obrazu w qemu

```sh
ob test .                  # jeśli w build/ jest jeden .iso - uruchamia go
ob test . --profile live   # wybór konkretnego obrazu, gdy jest ich kilka
```

Na hoście z zainstalowanym qemu obraz otwiera się w zwykłym oknie qemu
(SDL/GTK). Bez qemu na hoście - tak samo jak przy `ob build` - `ob test`
zapyta (albo w CI automatycznie wybierze) tryb kontenerowy: dedykowany
kontener podman z zainstalowanym qemu, którego ekran wystawiany jest przez
VNC na `127.0.0.1:5900` (port do zmiany flagą `--vnc-port`) - połącz się
dowolnym klientem VNC, żeby zobaczyć ekran maszyny wirtualnej.

To tylko szybki "czy się odpala" smoke test, nie zastępuje testowania na
prawdziwym sprzęcie.

## Schemat `config.yaml`

```yaml
distribution:
  name: "Orange Linux"
  version: "Tumbleweed"        # Tumbleweed | Leap | MicroOS
  leap_version: "15.6"         # wymagane tylko gdy version: Leap
  micro_os_token: ""           # wymagane tylko gdy version: MicroOS
  description: "..."
  license: "GPL-3.0"

image:
  name: "orange-linux"         # nazwa finalnego pliku ISO
  version: "1.0.0"
  filesystem: "btrfs"          # ext4 | btrfs | xfs
  arch: "x86_64"

desktop:
  environment: "kde"           # kde | gnome | xfce | none

installer:
  type: "calamares"            # calamares | none
  live_user:
    create: true
    username: "live"
    password: "live"
    autologin: true

repositories:
  - name: "oss"
    url: "http://download.opensuse.org/tumbleweed/repo/oss/"
  - name: "non-oss"
    url: "http://download.opensuse.org/tumbleweed/repo/non-oss/"

# Opcjonalne - patrz sekcje "Profile obrazu" i "Sumy kontrolne i podpis GPG" wyżej.
profiles:
  - name: "live"
    type: "iso"
    default: true
  - name: "disk"
    type: "oem"

signing:
  gpg_key_id: "TWOJ_ID_KLUCZA_GPG"
```

`MicroOS` wymaga podania `micro_os_token` (tokenu rejestracyjnego) -
zgodnie z założeniem, że warianty atomowe openSUSE wymagają dodatkowej
konfiguracji rejestracji.

## Architektura kodu źródłowego

```
main.go                  punkt wejścia, deleguje do src/cli
go.mod
Makefile
src/
├── cli/                  obsługa komend (build/test/clean/init/validate/help)
├── config/                parser YAML (bez zależności zewnętrznych) + walidacja
├── project/                odczyt packages/, rpm-files/, files/, hooks/, brandings/
├── kiwi/                   generowanie config.xml/config.sh dla kiwi-ng (w tym
│                           profili), pełnej konfiguracji Calamares (calamares.go),
│                           zarządzanie kontenerami podman do budowania i testowania
│                           (podman.go, qemu.go) i wywołanie `kiwi-ng system build`
├── release/                sumy kontrolne SHA256 + opcjonalny podpis GPG
└── util/                   logowanie, operacje na plikach, uruchamianie
                             poleceń, pytania tak/nie w terminalu (prompt.go)
```

### Uwaga o parserze YAML

Aby `go build` działał od razu, bez potrzeby pobierania zależności z
internetu, `src/config/yaml.go` zawiera własny, minimalny parser YAML
obsługujący podzbiór składni potrzebny dla `config.yaml` (mapy, listy,
proste skalary). Jeśli projekt zostanie podłączony do sieci / modułów Go,
można to w każdej chwili podmienić na `gopkg.in/yaml.v3` bez zmiany
reszty kodu - wystarczy zamienić implementację `ParseYAML`.

## Status / TODO

To jest działający szkielet architektury, przygotowany pod dalszy rozwój:
- [x] pełna konfiguracja Calamares (branding + settings.conf + moduły +
      automatyczne wpięcie joba `shellprocess` dla aplikacji flatpak)
- [x] automatyczny fallback do izolowanego kontenera podman z kiwi-ng,
      gdy hosta nie ma zainstalowanego kiwi-ng (interaktywne pytanie
      lub automatyczny wybór w CI + flagi `--container`/`--no-container`)
- [x] automatyczne odświeżanie kontenera podman gdy zmieni się wersja
      dystrybucji w config.yaml
- [x] trwały cache pakietów w `build/cache/` (tryb kontenerowy, best-effort)
- [x] sumy kontrolne SHA256 zawsze, opcjonalny podpis GPG (`signing:`)
- [x] wsparcie dla wielu profili obrazu (`profiles:`, `--profile`, `--all-profiles`)
- [x] obrazy brandingu z prostego katalogu `brandings/` w korzeniu projektu
- [x] `ob test` - szybkie uruchomienie zbudowanego ISO w qemu (host albo kontener + VNC)
- [ ] cache pakietów dla budowania NA HOŚCIE (bez kontenera) - obecnie
      dotyczy tylko trybu kontenerowego
- [ ] per-profilowe listy pakietów/repozytoriów (obecnie wspólne dla wszystkich profili)
- [ ] `ob test` nie sprawdza automatycznie czy VM w ogóle doszła do
      ekranu powitalnego/instalatora - to nadal ręczna obserwacja przez VNC
