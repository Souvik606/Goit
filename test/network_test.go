package test

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"souvik606/goit/pkg/goit/local"
	"souvik606/goit/pkg/goit/remote"
	"testing"
	"time"
)

func createTestCommit(t *testing.T, dir, file, content, message string) string {
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Failed to chdir to %s: %v", dir, err)
	}

	writeFile(t, file, content)
	addAll(t)
	hash, _, err := commit(t, message)
	if err != nil {
		t.Fatalf("Failed to commit %s: %v", message, err)
	}
	return hash
}

func TestCloneAndFetch(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get wd: %v", err)
	}
	defer os.Chdir(originalWd)

	// --- 1. Setup Server Repo ---
	serverRoot := t.TempDir()
	bareRepoPath := filepath.Join(serverRoot, "test-repo.git")
	err = os.Mkdir(bareRepoPath, 0755)
	if err != nil {
		t.Fatalf("Mkdir bare repo failed: %v", err)
	}

	// We need a temporary normal repo to create the first commit
	tempClonePath := filepath.Join(t.TempDir(), "temp-clone")

	err = os.Mkdir(tempClonePath, 0755)
	if err != nil {
		t.Fatalf("Mkdir temp clone failed: %v", err)
	}
	if err := local.InitRepository(tempClonePath, false); err != nil {
		t.Fatalf("Init temp clone failed: %v", err)
	}
	if err := os.Chdir(tempClonePath); err != nil {
		t.Fatalf("Chdir to temp clone failed: %v", err)
	}

	commitHashC1 := createTestCommit(t, tempClonePath, "fileA.txt", "contentA", "Commit 1")

	// Manually "push" by copying objects and updating refs in bare repo
	if err := local.InitRepository(bareRepoPath, true); err != nil {
		t.Fatalf("Init bare repo failed: %v", err)
	}
	copyObjectFiles(t, filepath.Join(tempClonePath, ".goit", "objects"), filepath.Join(bareRepoPath, "objects"))

	bareRefPath := filepath.Join(bareRepoPath, "refs", "heads", "main")
	if err := os.MkdirAll(filepath.Dir(bareRefPath), 0755); err != nil {
		t.Fatalf("Failed to create bare ref dirs: %v", err)
	}
	if err := os.WriteFile(bareRefPath, []byte(commitHashC1+"\n"), 0644); err != nil {
		t.Fatalf("Failed to update bare repo ref: %v", err)
	}

	// --- 2. Start Server ---
	// Need to run server logic in a goroutine
	serverPort := "8081" // Use a non-default port
	go func() {
		// This will block until the server is killed (which is fine for test)
		remote.Serve(serverRoot, ":"+serverPort)
	}()
	time.Sleep(50 * time.Millisecond) // Give server a moment to start

	// --- 3. Run Clone ---
	clientClonePath := filepath.Join(t.TempDir(), "client-clone")
	serverURL := "http://localhost:" + serverPort + "/test-repo.git"

	if err := os.Chdir(filepath.Dir(clientClonePath)); err != nil {
		t.Fatalf("Failed to chdir for clone: %v", err)
	}

	err = remote.GoitClone(serverURL, "client-clone")
	if err != nil {
		t.Fatalf("GoitClone failed: %v", err)
	}

	// --- 4. Verify Clone ---
	if err := os.Chdir(clientClonePath); err != nil {
		t.Fatalf("Failed to chdir into cloned repo: %v", err)
	}

	if _, err := os.Stat("fileA.txt"); err != nil {
		t.Errorf("Cloned repo missing fileA.txt: %v", err)
	}

	headHash, err := local.GetHeadCommitHash()
	if err != nil {
		t.Fatalf("Failed to get HEAD hash in cloned repo: %v", err)
	}
	if headHash != commitHashC1 {
		t.Errorf("Cloned repo HEAD hash mismatch. Got %s, want %s", headHash[:7], commitHashC1[:7])
	}

	// --- 5. Add New Commit to Server ---
	if err := os.Chdir(tempClonePath); err != nil {
		t.Fatalf("Failed to chdir back to temp clone: %v", err)
	}
	commitHashC2 := createTestCommit(t, tempClonePath, "fileB.txt", "contentB", "Commit 2")
	copyObjectFiles(t, filepath.Join(tempClonePath, ".goit", "objects"), filepath.Join(bareRepoPath, "objects"))

	if err := os.WriteFile(bareRefPath, []byte(commitHashC2+"\n"), 0644); err != nil { // #changed
		t.Fatalf("Failed to update bare repo ref for C2: %v", err) // #changed
	}

	// --- 6. Run Fetch from Client ---
	if err := os.Chdir(clientClonePath); err != nil {
		t.Fatalf("Failed to chdir back to client clone: %v", err)
	}

	_, err = remote.GoitFetch("origin")
	if err != nil {
		t.Fatalf("GoitFetch failed: %v", err)
	}

	// --- 7. Verify Fetch ---
	remoteMainHash, err := local.GetRefHash("refs/remotes/origin/main")
	if err != nil {
		t.Fatalf("Failed to read remote ref after fetch: %v", err)
	}
	if remoteMainHash != commitHashC2 {
		t.Errorf("Remote ref not updated by fetch. Got %s, want %s", remoteMainHash[:7], commitHashC2[:7])
	}

	localMainHash, err := local.GetRefHash("refs/heads/main")
	if err != nil {
		t.Fatalf("Failed to read local ref after fetch: %v", err)
	}
	if localMainHash != commitHashC1 {
		t.Errorf("Local main branch was modified by fetch. Got %s, want %s", localMainHash[:7], commitHashC1[:7])
	}
}

// Simple object copy, not efficient but works for test
func copyObjectFiles(t *testing.T, srcDir, dstDir string) {
	filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dstDir, relPath)

		if _, err := os.Stat(dstPath); err == nil {
			return nil
		}

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}
