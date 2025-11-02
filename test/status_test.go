package test

import (
	"os"
	"path/filepath"
	"souvik606/goit/pkg/goit"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	dir := filepath.Dir(path)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

func addAll(t *testing.T) {
	index := goit.NewIndex()
	err := index.Load()
	if err != nil {
		t.Fatalf("Failed to load index for addAll: %v", err)
	}
	err = goit.AddPaths([]string{"."}, index)
	if err != nil {
		t.Fatalf("Failed to addAll: %v", err)
	}
	err = index.Save()
	if err != nil {
		t.Fatalf("Failed to save index for addAll: %v", err)
	}
}

func commit(t *testing.T, message string) (string, string, error) {
	hash, ref, err := goit.Commit(message)
	if err != nil {
		t.Fatalf("Failed to commit '%s': %v", message, err)
	}
	return hash, ref, err
}

func TestStatusOnNewRepo(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	summary, err := goit.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if len(summary.Staged) != 0 || len(summary.Unstaged) != 0 || len(summary.Untracked) != 0 {
		t.Errorf("Expected clean status on new repo, but got: Staged(%d), Unstaged(%d), Untracked(%d)",
			len(summary.Staged), len(summary.Unstaged), len(summary.Untracked))
	}
}

func TestStatusUntracked(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "file.txt", "untracked")

	summary, err := goit.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if len(summary.Untracked) != 1 || summary.Untracked[0] != "file.txt" {
		t.Errorf("Expected 1 untracked file ('file.txt'), got: %v", summary.Untracked)
	}
	if len(summary.Staged) != 0 {
		t.Errorf("Staged list should be empty, got %d", len(summary.Staged))
	}
	if len(summary.Unstaged) != 0 {
		t.Errorf("Unstaged list should be empty, got %d", len(summary.Unstaged))
	}
}

func TestStatusStagedNew(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "file.txt", "staged")
	addAll(t)

	summary, err := goit.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if len(summary.Staged) != 1 || summary.Staged["file.txt"] != goit.ChangeStagedNew {
		t.Errorf("Expected 1 'StagedNew' file, got: %v", summary.Staged)
	}
	if len(summary.Unstaged) != 0 {
		t.Errorf("Unstaged list should be empty, got %d", len(summary.Unstaged))
	}
	if len(summary.Untracked) != 0 {
		t.Errorf("Untracked list should be empty, got %d", len(summary.Untracked))
	}
}

func TestStatusCommitted(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "file.txt", "committed")
	addAll(t)
	commit(t, "C1")

	summary, err := goit.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if len(summary.Staged) != 0 || len(summary.Unstaged) != 0 || len(summary.Untracked) != 0 {
		t.Errorf("Expected clean status after commit, but got: Staged(%d), Unstaged(%d), Untracked(%d)",
			len(summary.Staged), len(summary.Unstaged), len(summary.Untracked))
	}
}

func TestStatusModified(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "file.txt", "committed")
	addAll(t)
	commit(t, "C1")

	writeFile(t, "file.txt", "modified")

	summary, err := goit.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if len(summary.Unstaged) != 1 || summary.Unstaged["file.txt"] != goit.ChangeModified {
		t.Errorf("Expected 1 'Modified' file in Unstaged, got: %v", summary.Unstaged)
	}
	if len(summary.Staged) != 0 {
		t.Errorf("Staged list should be empty, got %d", len(summary.Staged))
	}
	if len(summary.Untracked) != 0 {
		t.Errorf("Untracked list should be empty, got %d", len(summary.Untracked))
	}
}

func TestStatusStagedModified(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "file.txt", "committed")
	addAll(t)
	commit(t, "C1")

	writeFile(t, "file.txt", "modified")
	addAll(t)

	summary, err := goit.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if len(summary.Staged) != 1 || summary.Staged["file.txt"] != goit.ChangeStagedModified {
		t.Errorf("Expected 1 'StagedModified' file, got: %v", summary.Staged)
	}
	if len(summary.Unstaged) != 0 {
		t.Errorf("Unstaged list should be empty, got %d", len(summary.Unstaged))
	}
}

func TestStatusDeleted(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "file.txt", "committed")
	addAll(t)
	commit(t, "C1")

	os.Remove("file.txt")

	summary, err := goit.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if len(summary.Unstaged) != 1 || summary.Unstaged["file.txt"] != goit.ChangeDeleted {
		t.Errorf("Expected 1 'Deleted' file in Unstaged, got: %v", summary.Unstaged)
	}
	if len(summary.Staged) != 0 {
		t.Errorf("Staged list should be empty, got %d", len(summary.Staged))
	}
}

func TestStatusStagedDeleted(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "file.txt", "committed")
	addAll(t)
	commit(t, "C1")

	os.Remove("file.txt")
	addAll(t)

	summary, err := goit.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if len(summary.Staged) != 1 || summary.Staged["file.txt"] != goit.ChangeStagedDeleted {
		t.Errorf("Expected 1 'StagedDeleted' file, got: %v", summary.Staged)
	}
	if len(summary.Unstaged) != 0 {
		t.Errorf("Unstaged list should be empty, got %d", len(summary.Unstaged))
	}
}

func TestStatusModifiedAndStaged(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "file.txt", "v1")
	addAll(t)
	commit(t, "C1")

	writeFile(t, "file.txt", "v2")
	addAll(t)

	writeFile(t, "file.txt", "v3")

	summary, err := goit.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %S", err)
	}

	if len(summary.Staged) != 1 || summary.Staged["file.txt"] != goit.ChangeStagedModified {
		t.Errorf("Expected 1 'StagedModified' (v2) file, got: %v", summary.Staged)
	}
	if len(summary.Unstaged) != 1 || summary.Unstaged["file.txt"] != goit.ChangeModified {
		t.Errorf("Expected 1 'Modified' (v3) file in Unstaged, got: %v", summary.Unstaged)
	}
}
