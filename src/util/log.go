package util

import "fmt"

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// Info wypisuje komunikat informacyjny (na stdout).
func Info(format string, a ...interface{}) {
	fmt.Printf(colorCyan+"[ob]"+colorReset+" "+format+"\n", a...)
}

// Step wypisuje nagłówek etapu budowania.
func Step(format string, a ...interface{}) {
	fmt.Printf(colorBold+colorGreen+"==> "+format+colorReset+"\n", a...)
}

// Warn wypisuje ostrzeżenie (na stdout, kolor żółty).
func Warn(format string, a ...interface{}) {
	fmt.Printf(colorYellow+"[ob] UWAGA:"+colorReset+" "+format+"\n", a...)
}

// Error wypisuje błąd (na stdout, kolor czerwony) - nie kończy programu.
func Error(format string, a ...interface{}) {
	fmt.Printf(colorRed+"[ob] BŁĄD:"+colorReset+" "+format+"\n", a...)
}
