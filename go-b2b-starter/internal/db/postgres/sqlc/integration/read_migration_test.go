package integration

import (
	"os"
	"path/filepath"
	"runtime"
)

// readMigration reads a migration file by name from the migrations directory.
func readMigration(name string) ([]byte, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(filepath.Join(filepath.Dir(thisFile), "..", "migrations", name))
}
