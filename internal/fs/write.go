package fs

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

// WriteFileAtomic replaces path with data, so a crash leaves the old
// file or the new one, never half of each.
func WriteFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err := f.Chmod(0o644); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return renameRetry(name, path)
}

// renameRetry retries the replace on Windows, where another process
// holding the destination open makes os.Rename fail with a sharing or
// access violation.
func renameRetry(old, new string) error {
	for attempt := 0; ; attempt++ {
		err := os.Rename(old, new)
		if err == nil {
			return nil
		}
		if runtime.GOOS != "windows" || attempt >= 9 {
			return err
		}
		if !errors.Is(err, syscall.Errno(32)) && !errors.Is(err, syscall.Errno(5)) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}
