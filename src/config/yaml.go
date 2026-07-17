package config

import (
	"fmt"
	"strings"
)

// Ten plik zawiera bardzo lekki parser YAML obsługujący jedynie podzbiór
// składni potrzebny do plików config.yaml Orange Buildera:
//   - zagnieżdżone mapy (klucz: wartość / klucz: <nowa linia z wcięciem>)
//   - listy ("- wartość" oraz "- klucz: wartość" z kontynuacją pól)
//   - proste skalary: napisy, true/false, null/~, wartości w cudzysłowach
//   - komentarze całych linii zaczynające się od '#'
//
// Celowo nie używamy zewnętrznej biblioteki (np. gopkg.in/yaml.v3), żeby
// `go build` działał od razu, bez dostępu do sieci / go.sum. Jeśli w
// przyszłości projekt zostanie podłączony do internetu, można to podmienić
// na pełnoprawny parser YAML bez zmiany reszty kodu (patrz LoadConfig).

type rawLine struct {
	indent int
	text   string
}

// ParseYAML parsuje zawartość pliku YAML do map[string]interface{}.
func ParseYAML(data []byte) (map[string]interface{}, error) {
	lines := tokenize(string(data))
	if len(lines) == 0 {
		return map[string]interface{}{}, nil
	}
	result, _, err := parseMap(lines, 0, lines[0].indent)
	return result, err
}

// tokenize zamienia surowy tekst na listę linii z policzonym wcięciem,
// pomijając linie puste oraz komentarze.
func tokenize(data string) []rawLine {
	var out []rawLine
	for _, raw := range strings.Split(data, "\n") {
		// Usuwamy końcowy CR (pliki zapisane w Windows).
		raw = strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := 0
		for indent < len(raw) && raw[indent] == ' ' {
			indent++
		}
		trimmed = stripInlineComment(trimmed)
		if trimmed == "" {
			continue
		}
		out = append(out, rawLine{indent: indent, text: trimmed})
	}
	return out
}

// stripInlineComment usuwa komentarz znajdujący się na końcu linii (np.
// `type: "calamares"   # calamares | none`), o ile '#' nie znajduje się
// wewnątrz cudzysłowu i jest poprzedzony białym znakiem (albo jest
// pierwszym znakiem linii - taki przypadek obsługuje już wcześniejsze
// sprawdzenie pełnych komentarzy w tokenize).
func stripInlineComment(s string) string {
	inQuote := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote != 0 {
			if c == inQuote {
				inQuote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			inQuote = c
			continue
		}
		if c == '#' && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t') {
			return strings.TrimSpace(s[:i])
		}
	}
	return s
}

// splitKeyValue dzieli linię "klucz: wartość" na klucz i wartość, ignorując
// dwukropki znajdujące się wewnątrz cudzysłowów.
func splitKeyValue(line string) (string, string, error) {
	inQuote := byte(0)
	for i := 0; i < len(line); i++ {
		c := line[i]
		if inQuote != 0 {
			if c == inQuote {
				inQuote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			inQuote = c
			continue
		}
		if c == ':' && (i == len(line)-1 || line[i+1] == ' ') {
			key := strings.TrimSpace(line[:i])
			val := ""
			if i+1 < len(line) {
				val = strings.TrimSpace(line[i+1:])
			}
			return key, val, nil
		}
	}
	return "", "", fmt.Errorf("nie znaleziono ':' w linii: %q", line)
}

// parseScalar konwertuje surowy tekst wartości na string/bool/nil.
// Wszystkie inne wartości (w tym liczby, np. wersje "15.6") są trzymane
// jako string - to celowe, bo w konfiguracji Orange Buildera są one i tak
// od razu odczytywane jako stringi.
func parseScalar(s string) interface{} {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	switch s {
	case "true", "yes", "on":
		return true
	case "false", "no", "off":
		return false
	case "null", "~", "":
		return nil
	}
	return s
}

// parseMap parsuje mapę klucz-wartość zaczynając od lines[idx], zakładając,
// że wszystkie klucze na tym poziomie mają wcięcie równe indent.
// Zwraca zbudowaną mapę oraz indeks pierwszej linii, która nie należy już
// do tego bloku.
func parseMap(lines []rawLine, idx int, indent int) (map[string]interface{}, int, error) {
	result := map[string]interface{}{}
	for idx < len(lines) && lines[idx].indent == indent && !strings.HasPrefix(lines[idx].text, "- ") && lines[idx].text != "-" {
		key, rest, err := splitKeyValue(lines[idx].text)
		if err != nil {
			return nil, idx, fmt.Errorf("błąd parsowania YAML: %w", err)
		}
		idx++
		if rest != "" {
			result[key] = parseScalar(rest)
			continue
		}
		// Wartość pusta - sprawdzamy czy dalej jest zagnieżdżony blok.
		if idx < len(lines) && lines[idx].indent > indent {
			childIndent := lines[idx].indent
			if strings.HasPrefix(lines[idx].text, "- ") || lines[idx].text == "-" {
				list, nidx, err := parseList(lines, idx, childIndent)
				if err != nil {
					return nil, idx, err
				}
				result[key] = list
				idx = nidx
			} else {
				m, nidx, err := parseMap(lines, idx, childIndent)
				if err != nil {
					return nil, idx, err
				}
				result[key] = m
				idx = nidx
			}
		} else {
			result[key] = nil
		}
	}
	return result, idx, nil
}

// parseList parsuje listę zaczynającą się od lines[idx], zakładając że
// pozycje listy ("- ...") mają wcięcie równe indent.
func parseList(lines []rawLine, idx int, indent int) ([]interface{}, int, error) {
	var result []interface{}
	for idx < len(lines) && lines[idx].indent == indent && (strings.HasPrefix(lines[idx].text, "- ") || lines[idx].text == "-") {
		content := strings.TrimSpace(strings.TrimPrefix(lines[idx].text, "-"))
		idx++
		if content == "" {
			// Element listy to zagnieżdżony blok (mapa) w kolejnych liniach.
			if idx < len(lines) && lines[idx].indent > indent {
				m, nidx, err := parseMap(lines, idx, lines[idx].indent)
				if err != nil {
					return nil, idx, err
				}
				result = append(result, m)
				idx = nidx
			} else {
				result = append(result, nil)
			}
			continue
		}
		if looksLikeKeyValue(content) {
			// "- klucz: wartość" - element listy jest mapą, ewentualne
			// kolejne pola tej mapy są wcięte o 2 znaki więcej niż "-".
			key, val, err := splitKeyValue(content)
			if err != nil {
				return nil, idx, err
			}
			item := map[string]interface{}{}
			if val != "" {
				item[key] = parseScalar(val)
			} else {
				item[key] = nil
			}
			itemIndent := indent + 2
			if idx < len(lines) && lines[idx].indent == itemIndent {
				rest, nidx, err := parseMap(lines, idx, itemIndent)
				if err != nil {
					return nil, idx, err
				}
				for k, v := range rest {
					item[k] = v
				}
				idx = nidx
			}
			result = append(result, item)
			continue
		}
		result = append(result, parseScalar(content))
	}
	return result, idx, nil
}

// looksLikeKeyValue sprawdza czy zawartość wygląda jak "klucz: wartość"
// (a nie jak zwykły skalar, np. nazwa pakietu albo URL).
func looksLikeKeyValue(content string) bool {
	_, _, err := splitKeyValue(content)
	return err == nil
}
