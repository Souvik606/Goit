package remote

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type GoitServer struct {
	BasePath string
}

func (s *GoitServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("DEBUG(Server): Received request: %s %s\n", r.Method, r.URL.Path)

	parts := strings.SplitN(strings.Trim(r.URL.Path, "/"), "/", 2)
	if len(parts) < 1 {
		http.Error(w, "Invalid request path", http.StatusBadRequest)
		return
	}

	repoName := parts[0]
	repoPath := filepath.Join(s.BasePath, repoName)

	if !isValidBareRepo(repoPath) {
		http.Error(w, "Repository not found or not a bare repository", http.StatusNotFound)
		return
	}

	if len(parts) == 1 {
		fmt.Fprintf(w, "Welcome to the goit repo: %s", repoName)
		return
	}

	action := parts[1]

	switch action {
	case "info/refs":
		fmt.Fprintf(w, "TODO: Implement info/refs for %s", repoName)
	case "get-objects":
		fmt.Fprintf(w, "TODO: Implement get-objects for %s", repoName)
	default:
		http.Error(w, "Action not supported", http.StatusNotFound)
	}
}

func isValidBareRepo(path string) bool {
	headPath := filepath.Join(path, "HEAD")
	if _, err := os.Stat(headPath); os.IsNotExist(err) {
		return false
	}
	configPath := filepath.Join(path, "config")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false
	}
	objectsPath := filepath.Join(path, "objects")
	if _, err := os.Stat(objectsPath); os.IsNotExist(err) {
		return false
	}
	return true
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
