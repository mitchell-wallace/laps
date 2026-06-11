package store

import (
	"os"
	"path/filepath"
	"strings"
)

func ClaimPath(beadsDir string) string {
	return filepath.Join(beadsDir, "claim")
}

func ReadClaim(beadsDir string) (string, error) {
	path := ClaimPath(beadsDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func WriteClaim(beadsDir, id string) error {
	path := ClaimPath(beadsDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return SafeWriteFile(path, []byte(id+"\n"), 0o644)
}

func RemoveClaim(beadsDir string) error {
	path := ClaimPath(beadsDir)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
