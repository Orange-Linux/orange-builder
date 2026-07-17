package release

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"orangebuilder/src/config"
	"orangebuilder/src/util"
)

// Finalize liczy sumę SHA256 gotowego obrazu ISO i zapisuje ją obok niego
// (plik "<iso>.sha256", format zgodny z `sha256sum -c`). Jeśli w
// config.yaml podano signing.gpg_key_id i polecenie `gpg` jest dostępne
// na hoście, dodatkowo tworzy odłączony podpis "<iso>.asc". Brak gpg lub
// brak sekcji signing NIE jest błędem - podpisywanie jest w pełni
// opcjonalne, suma kontrolna jest liczona zawsze.
func Finalize(isoPath string, cfg *config.Config) error {
	sum, err := sha256File(isoPath)
	if err != nil {
		return fmt.Errorf("nie udało się policzyć sumy sha256 obrazu: %w", err)
	}

	sumPath := isoPath + ".sha256"
	line := fmt.Sprintf("%s  %s\n", sum, filepath.Base(isoPath))
	if err := os.WriteFile(sumPath, []byte(line), 0o644); err != nil {
		return fmt.Errorf("nie udało się zapisać %s: %w", sumPath, err)
	}
	util.Info("Suma SHA256 zapisana w: %s", sumPath)

	if cfg.Signing.GPGKeyID == "" {
		return nil
	}
	if !util.CommandExists("gpg") {
		util.Warn("skonfigurowano signing.gpg_key_id, ale nie znaleziono polecenia \"gpg\" na hoście - pomijam podpisywanie obrazu")
		return nil
	}

	ascPath := isoPath + ".asc"
	err = util.RunCommand("", "gpg",
		"--batch", "--yes",
		"--local-user", cfg.Signing.GPGKeyID,
		"--detach-sign", "--armor",
		"--output", ascPath,
		isoPath,
	)
	if err != nil {
		return fmt.Errorf("podpisywanie GPG obrazu nie powiodło się: %w", err)
	}
	util.Info("Podpis GPG zapisany w: %s", ascPath)
	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
