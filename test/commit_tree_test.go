package test

import (
	"fmt"
	"os"
	goit "souvik606/goit/pkg/goit/local"
	"strings"
	"testing"
)

func TestCommitTree(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	emptyTreeHash := "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	emptyTreeData := goit.FormatObject("tree", []byte{})
	goit.WriteObject(emptyTreeHash, emptyTreeData)

	parentHash := "0123456789abcdef0123456789abcdef01234567"
	message := "Test initial commit\n\nWith a body."
	expectedAuthor := "Test Author <author@test.com>"
	expectedCommitter := "Test Committer <committer@test.com>"

	os.Setenv("GOIT_AUTHOR_NAME", "Test Author")
	os.Setenv("GOIT_AUTHOR_EMAIL", "author@test.com")

	os.Setenv("GOIT_COMMITTER_NAME", "Test Committer")
	os.Setenv("GOIT_COMMITTER_EMAIL", "committer@test.com")

	defer os.Unsetenv("GOIT_AUTHOR_NAME")
	defer os.Unsetenv("GOIT_AUTHOR_EMAIL")
	defer os.Unsetenv("GOIT_COMMITTER_NAME")
	defer os.Unsetenv("GOIT_COMMITTER_EMAIL")

	commitHash, err := goit.CommitTree(emptyTreeHash, []string{parentHash}, message)
	if err != nil {
		t.Fatalf("CommitTree failed: %v", err)
	}

	if len(commitHash) != 40 {
		t.Errorf("Expected a 40-char hash, got %d chars: %s", len(commitHash), commitHash)
	}

	objType, contentBytes, err := goit.CatFile(commitHash)
	if err != nil {
		t.Fatalf("CatFile failed for commit hash %s: %v", commitHash, err)
	}
	if objType != "commit" {
		t.Errorf("Expected object type 'commit', got '%s'", objType)
	}

	content := string(contentBytes)

	if !strings.Contains(content, fmt.Sprintf("tree %s\n", emptyTreeHash)) {
		t.Errorf("Commit missing correct tree hash line")
	}
	if !strings.Contains(content, fmt.Sprintf("parent %s\n", parentHash)) {
		t.Errorf("Commit missing correct parent hash line")
	}
	if !strings.Contains(content, fmt.Sprintf("author %s", expectedAuthor)) {
		t.Errorf("Commit missing correct author line")
	}
	if !strings.Contains(content, fmt.Sprintf("committer %s", expectedCommitter)) {
		t.Errorf("Commit missing correct committer line")
	}

	trimmedMessage := strings.TrimSpace(message)
	if !strings.Contains(content, "\n\n"+trimmedMessage) {
		t.Errorf("Commit missing correct message. Got:\n---\n%s\n---\nExpected message:\n---\n%s\n---\n", content, trimmedMessage)
	}
}

func TestCommitTreeNoParent(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	emptyTreeHash := "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	emptyTreeData := goit.FormatObject("tree", []byte{})
	goit.WriteObject(emptyTreeHash, emptyTreeData)

	os.Setenv("GOIT_AUTHOR_NAME", "Test Author")
	os.Setenv("GOIT_AUTHOR_EMAIL", "author@test.com")
	os.Setenv("GOIT_COMMITTER_NAME", "Test Author")
	os.Setenv("GOIT_COMMITTER_EMAIL", "author@test.com")
	defer os.Unsetenv("GOIT_AUTHOR_NAME")
	defer os.Unsetenv("GOIT_AUTHOR_EMAIL")
	defer os.Unsetenv("GOIT_COMMITTER_NAME")
	defer os.Unsetenv("GOIT_COMMITTER_EMAIL")

	message := "Initial commit (no parent)"

	commitHash, err := goit.CommitTree(emptyTreeHash, nil, message)
	if err != nil {
		t.Fatalf("CommitTree failed: %v", err)
	}

	_, contentBytes, err := goit.CatFile(commitHash)
	if err != nil {
		t.Fatalf("CatFile failed for commit hash %s: %v", commitHash, err)
	}
	content := string(contentBytes)

	if strings.Contains(content, "parent ") {
		t.Errorf("Initial commit should not contain a parent line")
	}
	if !strings.Contains(content, fmt.Sprintf("tree %s\n", emptyTreeHash)) {
		t.Errorf("Commit missing correct tree hash line")
	}
}
