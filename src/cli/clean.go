package cli

import (
	"os"
	"path/filepath"

	"orangebuilder/src/kiwi"
	"orangebuilder/src/util"
)

type cleanOptions struct {
	projectDir string
	all        bool // --all: usuń też build/cache/ (trwały cache pakietów)
}

func parseCleanArgs(args []string) cleanOptions {
	opts := cleanOptions{projectDir: "."}
	for _, a := range args {
		switch a {
		case "--all":
			opts.all = true
		default:
			opts.projectDir = a
		}
	}
	return opts
}

func runClean(args []string) error {
	opts := parseCleanArgs(args)
	absProjectDir, err := filepath.Abs(opts.projectDir)
	if err != nil {
		return err
	}
	buildDir := filepath.Join(absProjectDir, "build")

	if err := kiwi.RemoveContainer(absProjectDir); err != nil {
		util.Warn("nie udało się usunąć kontenera podman (budowanie): %s", err.Error())
	}
	if err := kiwi.RemoveTestContainer(absProjectDir); err != nil {
		util.Warn("nie udało się usunąć kontenera podman (test): %s", err.Error())
	}

	if !util.FileExists(buildDir) {
		util.Info("Katalog %s już nie istnieje - nic do wyczyszczenia.", buildDir)
		return nil
	}

	if opts.all {
		util.Step("Usuwanie %s (razem z cache pakietów)", buildDir)
		if err := os.RemoveAll(buildDir); err != nil {
			return err
		}
		util.Info("Wyczyszczono całkowicie (łącznie z build/cache/).")
		return nil
	}

	util.Step("Usuwanie %s (zachowuję build/%s/ - trwały cache pakietów)", buildDir, kiwi.CacheDirName)
	entries, err := os.ReadDir(buildDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() == kiwi.CacheDirName {
			continue
		}
		if err := os.RemoveAll(filepath.Join(buildDir, e.Name())); err != nil {
			return err
		}
	}
	util.Info("Wyczyszczono (cache pakietów zachowany - użyj `ob clean --all`, żeby usunąć też cache).")
	return nil
}
