package launchericon

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteIcon(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "launcher-icon")
	for _, on := range []bool{false, true, true, false} {
		if err := writeIcon(dir, on); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(dir, "icon.png"))
		if err != nil {
			t.Fatal(err)
		}
		want := offIcon
		if on {
			want = onIcon
		}
		if !bytes.Equal(got, want) {
			t.Fatal("icon does not match LED state")
		}
		if len(got) > 1<<20 {
			t.Fatal("icon exceeds platform limit")
		}
		if _, err := png.Decode(bytes.NewReader(got)); err != nil {
			t.Fatal(err)
		}
	}
	before, _ := os.Stat(filepath.Join(dir, "icon.png"))
	if err := writeIcon(dir, false); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(filepath.Join(dir, "icon.png"))
	if !os.SameFile(before, after) {
		t.Fatal("unchanged icon was replaced")
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := writeIcon(dir, true); err != nil {
		t.Fatal("restart regeneration:", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "icon.png" {
		t.Fatal("temporary files leaked", entries)
	}
}

func TestWriteIconFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeIcon(path, true); err == nil {
		t.Fatal("expected error for a non-directory")
	}
	got, _ := os.ReadFile(path)
	if string(got) != "keep" {
		t.Fatal("existing file changed")
	}
}
