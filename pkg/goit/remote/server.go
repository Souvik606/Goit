package remote

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"souvik606/goit/pkg/goit/local"
	"strings"
)

const objectsDir = "objects"

var (
	isHexHash = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

type GoitServer struct {
	BasePath string
}

type InfoRefsResponse struct {
	Head string            `json:"head"`
	Refs map[string]string `json:"refs"`
}

type GetObjectsRequest struct {
	Wants []string `json:"wants"`
	Haves []string `json:"haves"`
}

func (s *GoitServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.Trim(r.URL.Path, "/"), "/", 2)
	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "Invalid request path", http.StatusBadRequest)
		return
	}

	repoName := parts[0]
	repoPath := filepath.Join(s.BasePath, repoName)

	if !local.IsValidBareRepo(repoPath) {
		http.Error(w, "Repository not found or not a bare repository", http.StatusNotFound)
		return
	}

	if len(parts) == 1 {
		fmt.Fprintf(w, "Welcome to the goit repo: %s", repoName)
		return
	}

	action := parts[1]

	originalWd, err := os.Getwd()
	if err != nil {
		http.Error(w, "Server internal error (cwd)", http.StatusInternalServerError)
		return
	}
	if err := os.Chdir(repoPath); err != nil {
		http.Error(w, "Server internal error (chdir)", http.StatusInternalServerError)
		return
	}
	defer os.Chdir(originalWd)

	switch action {
	case "info/refs":
		s.handleInfoRefs(w, r)
	case "get-objects":
		s.handleGetObjects(w, r)
	case "receive-pack":
		handleReceivePack(w, r, repoPath)
	default:
		http.Error(w, "Action not supported", http.StatusNotFound)
	}
}

func (s *GoitServer) handleInfoRefs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	branches, _, err := local.ListBranches()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list branches: %v", err), http.StatusInternalServerError)
		return
	}

	headRef, err := local.GetHeadRef()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read HEAD: %v", err), http.StatusInternalServerError)
		return
	}

	headFileContent := ""
	if isHexHash.MatchString(headRef) {
		headFileContent = headRef
	} else {
		headFileContent = "ref: " + headRef
	}

	refsMap := make(map[string]string)
	for _, branch := range branches {
		refPath := "refs/heads/" + branch
		hash, err := local.GetRefHash(refPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read ref %s: %v", refPath, err), http.StatusInternalServerError)
			return
		}
		if hash != "" {
			refsMap[refPath] = hash
		}
	}

	resp := InfoRefsResponse{
		Head: headFileContent,
		Refs: refsMap,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Printf("Failed to encode response (client likely disconnected): %v\n", err)
	}
}

func (s *GoitServer) handleGetObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GetObjectsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	commitHashes, err := FindCommitsToSync(req.Wants, req.Haves)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed finding commits: %v", err), http.StatusInternalServerError)
		return
	}

	objectHashes, err := FindRequiredObjects(commitHashes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed finding objects: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-tar-gzip")

	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	for hash := range objectHashes {
		objectPath := local.GetObjectPath(hash)
		file, err := os.Open(objectPath)
		if err != nil {
			continue
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			continue
		}

		hdr := &tar.Header{
			Name: path.Join(hash[:2], hash[2:]),
			Size: stat.Size(),
			Mode: 0644,
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return
		}
		if _, err := io.Copy(tw, file); err != nil {
			return
		}
	}
}

func handleReceivePack(w http.ResponseWriter, r *http.Request, repoPath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	refName := r.URL.Query().Get("ref")
	oldHash := r.URL.Query().Get("old")
	newHash := r.URL.Query().Get("new")

	if refName == "" || newHash == "" {
		http.Error(w, "Missing 'ref' or 'new' parameters", http.StatusBadRequest)
		return
	}

	currentHash, err := local.ResolveRef(filepath.Join(repoPath, ".goit"), refName)
	if err != nil {
		if !os.IsNotExist(err) && oldHash != "" && oldHash != "0000000000000000000000000000000000000000" {
			// In a real system, we'd error here. For this simplified version, we proceed if it's a new branch.
		}
	} else {
		if currentHash != oldHash {
			http.Error(w, fmt.Sprintf("Rejecting push: remote ref has changed (expected %s, got %s)", oldHash, currentHash), http.StatusConflict)
			return
		}
	}

	gr, err := gzip.NewReader(r.Body)
	if err != nil {
		http.Error(w, "Failed to create gzip reader", http.StatusBadRequest)
		return
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	objectsDir := filepath.Join(repoPath, "objects")

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "Failed to read tar stream", http.StatusInternalServerError)
			return
		}

		targetPath := filepath.Join(objectsDir, header.Name)
		if !strings.HasPrefix(targetPath, filepath.Clean(objectsDir)) {
			http.Error(w, "Illegal file path in tar", http.StatusBadRequest)
			return
		}

		if header.Typeflag == tar.TypeReg {
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				http.Error(w, "Failed to create object directory", http.StatusInternalServerError)
				return
			}

			f, err := os.Create(targetPath)
			if err != nil {
				http.Error(w, "Failed to write object file", http.StatusInternalServerError)
				return
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				http.Error(w, "Failed to write object content", http.StatusInternalServerError)
				return
			}
			f.Close()
		}
	}

	if err := local.UpdateRefRaw(repoPath, refName, newHash); err != nil {
		http.Error(w, "Failed to update ref: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Push accepted: %s updated to %s", refName, newHash)
}

func Serve(basePath string, port string) error {
	server := &GoitServer{
		BasePath: basePath,
	}

	mux := http.NewServeMux()
	mux.Handle("/", server)
	fmt.Printf("Starting goit server for path %s on port %s...\n", basePath, port)
	return http.ListenAndServe(port, mux)
}
