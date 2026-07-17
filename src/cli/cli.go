package cli

import (
	"orangebuilder/src/util"
)

// Version to aktualna wersja narzędzia Orange Builder.
const Version = "0.1.0"

// Execute jest głównym punktem wejścia CLI - parsuje nazwę komendy i
// deleguje wykonanie do odpowiedniej funkcji. Zwraca kod wyjścia procesu.
func Execute(args []string) int {
	if len(args) == 0 {
		printHelp()
		return 1
	}

	cmd := args[0]
	rest := args[1:]

	var err error
	switch cmd {
	case "build":
		err = runBuild(rest)
	case "test":
		err = runTest(rest)
	case "clean":
		err = runClean(rest)
	case "init":
		err = runInit(rest)
	case "validate":
		err = runValidate(rest)
	case "version", "--version", "-v":
		printVersion()
		return 0
	case "help", "--help", "-h":
		printHelp()
		return 0
	default:
		util.Error("nieznana komenda: %q", cmd)
		printHelp()
		return 1
	}

	if err != nil {
		util.Error("%s", err.Error())
		return 1
	}
	return 0
}

// projectPathArg wyciąga opcjonalny argument ścieżki do projektu z listy
// argumentów podkomendy (np. `ob build /sciezka/do/projektu`). Jeśli
// argument nie jest podany, zwracana jest "." (bieżący katalog roboczy).
func projectPathArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "."
}
