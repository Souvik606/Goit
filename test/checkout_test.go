package test

import (
	"os"
	goit "souvik606/goit/pkg/goit/local"
	"strings"
	"testing"
)

func TestCheckoutBranch(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "fileA.txt", "v1")
	addAll(t)
	commit(t, "C1")

	err := goit.CreateBranch("feature")
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	writeFile(t, "fileB.txt", "v2")
	addAll(t)
	commit(t, "C2")

	_, err = goit.Checkout("feature")
	if err != nil {
		t.Fatalf("Checkout 'feature' failed: %v", err)
	}

	headRef, _ := goit.GetHeadRef()
	if headRef != "refs/heads/feature" {
		t.Errorf("HEAD ref not updated. Expected 'refs/heads/feature', got '%s'", headRef)
	}

	if _, err := os.Stat("fileA.txt"); err != nil {
		t.Errorf("fileA.txt should exist on 'feature' branch but does not")
	}

	if _, err := os.Stat("fileB.txt"); err == nil {
		t.Errorf("fileB.txt should NOT exist on 'feature' branch but does")
	}

	index := goit.NewIndex()
	index.Load()
	if len(index.Entries) != 1 {
		t.Errorf("Index should have 1 entry on 'feature', got %d", len(index.Entries))
	}
	if _, ok := index.Entries["fileA.txt"]; !ok {
		t.Errorf("Index missing 'fileA.txt' on 'feature' branch")
	}

	_, err = goit.Checkout("main")
	if err != nil {
		t.Fatalf("Checkout 'main' failed: %v", err)
	}

	headRef, _ = goit.GetHeadRef()
	if headRef != "refs/heads/main" {
		t.Errorf("HEAD ref not updated. Expected 'refs/heads/main', got '%s'", headRef)
	}

	if _, err := os.Stat("fileA.txt"); err != nil {
		t.Errorf("fileA.txt should exist on 'main' branch but does not")
	}

	if _, err := os.Stat("fileB.txt"); err != nil {
		t.Errorf("fileB.txt should exist on 'main' branch but does not")
	}

	index.Load()
	if len(index.Entries) != 2 {
		t.Errorf("Index should have 2 entries on 'main', got %d", len(index.Entries))
	}
}

func TestCheckoutDetachedHead(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "fileA.txt", "v1")
	addAll(t)
	commitHashC1, _, _ := commit(t, "C1")

	writeFile(t, "fileB.txt", "v2")
	addAll(t)
	commit(t, "C2")

	_, err := goit.Checkout(commitHashC1)
	if err != nil {
		t.Fatalf("Checkout commit hash failed: %v", err)
	}

	headRef, _ := goit.GetHeadRef()
	if headRef != commitHashC1 {
		t.Errorf("HEAD not in detached state. Expected '%s', got '%s'", commitHashC1, headRef)
	}

	if _, err := os.Stat("fileA.txt"); err != nil {
		t.Errorf("fileA.txt should exist on commit C1 but does not")
	}

	if _, err := os.Stat("fileB.txt"); err == nil {
		t.Errorf("fileB.txt should NOT exist on commit C1 but does")
	}
}

func TestCheckoutSafety(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "fileA.txt", "v1")
	addAll(t)
	commit(t, "C1")
	goit.CreateBranch("feature")

	writeFile(t, "fileA.txt", "v1-modified")

	_, err := goit.Checkout("feature")
	if err == nil {
		t.Fatalf("Checkout should have failed due to unstaged changes, but did not")
	}

	if !strings.Contains(err.Error(), "overwritten") || !strings.Contains(err.Error(), "fileA.txt") {
		t.Errorf("Expected unstaged changes error for 'fileA.txt', got: %v", err)
	}

	headRef, _ := goit.GetHeadRef()
	if headRef != "refs/heads/main" {
		t.Errorf("HEAD should not have changed. Expected 'refs/heads/main', got '%s'", headRef)
	}
}
