package test

import (
	"bytes"
	"io"
	"os"
	goit "souvik606/goit/pkg/goit/local"
	"strings"
	"testing"
)

func TestDiffWorkspaceIndex(t *testing.T) {
	// Re-using your existing test setup
	cleanup := setupTestRepo(t)
	defer cleanup()

	fileName := "diff_test.txt"
	initialContent := "line 1\nline 2\nline 3"
	modifiedContent := "line 1\nline 2 modified\nline 3\nline 4 added"

	// 1. Create the initial file and stage it
	err := os.WriteFile(fileName, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial test file: %v", err)
	}

	index := goit.NewIndex()
	err = goit.AddPaths([]string{fileName}, index)
	if err != nil {
		t.Fatalf("AddPaths failed: %v", err)
	}
	err = index.Save()
	if err != nil {
		t.Fatalf("Save index failed: %v", err)
	}

	// 2. Modify the file in the workspace (do not stage it)
	err = os.WriteFile(fileName, []byte(modifiedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// 3. Intercept os.Stdout to capture the diff output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the diff function
	err = goit.DiffWorkspaceIndex()

	// 4. Restore os.Stdout and read the captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("DiffWorkspaceIndex returned an error: %v", err)
	}

	// 5. Verify the algorithm correctly identified the insertions and deletions
	// Note: We check for the raw strings because the output includes ANSI color codes
	if !strings.Contains(output, "- line 2") {
		t.Errorf("Expected diff to show deletion of 'line 2', but it didn't.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, "+ line 2 modified") {
		t.Errorf("Expected diff to show addition of 'line 2 modified', but it didn't.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, "+ line 4 added") {
		t.Errorf("Expected diff to show addition of 'line 4 added', but it didn't.\nOutput:\n%s", output)
	}
}
