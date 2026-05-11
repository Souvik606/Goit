package test

import (
	"os"
	"path/filepath"
	"testing"

	"souvik606/goit/pkg/goit/local"
)

func TestRm(t *testing.T) {
	d := t.TempDir()
	cwd, _ := os.Getwd()
	os.Chdir(d)
	defer os.Chdir(cwd)

	if err := local.InitRepository(".", false); err != nil {
		t.Fatalf("err: %v", err)
	}

	f1 := "1.txt"
	os.WriteFile(f1, []byte("v1"), 0644)

	f2 := "2.txt"
	os.WriteFile(f2, []byte("v2"), 0644)

	dir := "logs"
	os.MkdirAll(dir, 0755)
	f3 := filepath.Join(dir, "3.log")
	os.WriteFile(f3, []byte("v3"), 0644)

	idx := local.NewIndex()
	idx.Load()
	local.AddPaths([]string{"."}, idx)
	idx.Save()

	t.Run("StdRm", func(t *testing.T) {
		err := local.Rm([]string{f1}, false, false)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		if _, err := os.Stat(f1); !os.IsNotExist(err) {
			t.Errorf("fail: %s exists", f1)
		}

		idx.Load()
		if _, ok := idx.Entries[f1]; ok {
			t.Errorf("fail: %s in idx", f1)
		}
	})

	t.Run("CachedRm", func(t *testing.T) {
		err := local.Rm([]string{f2}, true, false)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		if _, err := os.Stat(f2); os.IsNotExist(err) {
			t.Errorf("fail: %s missing", f2)
		}

		idx.Load()
		if _, ok := idx.Entries[f2]; ok {
			t.Errorf("fail: %s in idx", f2)
		}
	})

	t.Run("RecRm", func(t *testing.T) {
		err := local.Rm([]string{dir}, false, true)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		if _, err := os.Stat(f3); !os.IsNotExist(err) {
			t.Errorf("fail: %s exists", f3)
		}

		idx.Load()
		if _, ok := idx.Entries[filepath.ToSlash(f3)]; ok {
			t.Errorf("fail: %s in idx", f3)
		}
	})
}
