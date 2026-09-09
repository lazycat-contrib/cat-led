// Package launchericon synchronizes the deployment launcher icon with the device LED.
package launchericon

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

//go:embed on.png
var onIcon []byte

//go:embed off.png
var offIcon []byte

var mu sync.Mutex

// Update publishes a complete PNG only when its content changes. The runtime
// directory is recreated after app restarts; failures do not undo LED changes.
func Update(on bool) error {
	return writeIcon("/lzcapp/run/launcher-icon", on)
}

func writeIcon(dir string, on bool) error {
	mu.Lock()
	defer mu.Unlock()
	data := offIcon
	if on {
		data = onIcon
	}
	target := filepath.Join(dir, "icon.png")
	if current, err := os.ReadFile(target); err == nil && bytes.Equal(current, data) {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create launcher icon directory: %w", err)
	}
	f, err := os.CreateTemp(dir, ".icon-*.png")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), target)
}
