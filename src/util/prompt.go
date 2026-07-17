package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// IsInteractiveStdin sprawdza czy standardowe wejście jest terminalem
// (TTY), czy raczej np. potokiem/CI (wtedy nie wolno czekać na input
// użytkownika - trzeba podjąć decyzję automatycznie).
func IsInteractiveStdin() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// Confirm zadaje użytkownikowi pytanie tak/nie w terminalu i zwraca jego
// odpowiedź. defaultYes określa co się stanie gdy użytkownik po prostu
// wciśnie Enter bez wpisywania niczego. Akceptowane odpowiedzi (bez
// rozróżniania wielkości liter): t/tak/y/yes oraz n/nie/no.
func Confirm(prompt string, defaultYes bool) bool {
	suffix := "[t/N]"
	if defaultYes {
		suffix = "[T/n]"
	}
	fmt.Printf("%s %s: ", prompt, suffix)

	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))

	switch answer {
	case "":
		return defaultYes
	case "t", "tak", "y", "yes":
		return true
	case "n", "nie", "no":
		return false
	default:
		return defaultYes
	}
}
