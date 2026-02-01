package remote

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	fmt.Printf("DEBUG(Server): Received request: %s %s\n", r.Method, r.URL.Path)

	parts := strings.SplitN(strings.Trim(r.URL.Path, "/"), "/", 2)
	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "Invalid request path", http.StatusBadRequest)
		return
	}

	repoName := parts[0]
	repoPath := filepath.Join(s.BasePath, repoName)

	if !local.IsValidBareRepo(repoPath) {
		fmt.Printf("DEBUG(Server): Repo not found or not bare: %s\n", repoPath)
		http.Error(w, "Repository not found or not a bare repository", http.StatusNotFound)
		return
	}

	if len(parts) == 1 {
		fmt.Fprintf(w, "Welcome to the goit repo: %s", repoName)
		return
	}

	action := parts[1]
	fmt.Printf("DEBUG(Server): Routing action '%s' for repo '%s'\n", action, repoName)

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
		fmt.Printf("DEBUG(Server): Failed to encode response (client likely disconnected): %v\n", err) // #changed
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
			fmt.Printf("DEBUG(Server): Could not open object %s: %v\n", objectPath, err)
			continue
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			fmt.Printf("DEBUG(Server): Could not stat object %s: %v\n", objectPath, err)
			continue
		}

		hdr := &tar.Header{
			Name: filepath.Join(hash[:2], hash[2:]),
			Size: stat.Size(),
			Mode: 0644,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			fmt.Printf("DEBUG(Server): Failed to write tar header for %s: %v\n", hash, err)
			return
		}
		if _, err := io.Copy(tw, file); err != nil {
			fmt.Printf("DEBUG(Server): Failed to write object %s to tar: %v\n", hash, err)
			return
		}
	}
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
