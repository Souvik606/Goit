package remote

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"souvik606/goit/pkg/goit/local"
	"strings"
)

func FetchInfoRefs(url string) (*InfoRefsResponse, error) {
	resp, err := http.Get(url + "/info/refs")
	if err != nil {
		return nil, fmt.Errorf("http get %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d: %s", resp.StatusCode, resp.Status)
	}

	var infoRefs InfoRefsResponse
	if err := json.NewDecoder(resp.Body).Decode(&infoRefs); err != nil {
		return nil, fmt.Errorf("decoding info/refs response: %w", err)
	}

	return &infoRefs, nil
}

func FetchObjects(url string, req GetObjectsRequest) (io.ReadCloser, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling get-objects request: %w", err)
	}

	resp, err := http.Post(url+"/get-objects", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("http post %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("http status %d: %s", resp.StatusCode, resp.Status)
	}

	return resp.Body, nil
}

func UnpackObjects(tarballStream io.ReadCloser) error {
	defer tarballStream.Close()

	gzipReader, err := gzip.NewReader(tarballStream)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar header: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		destPath := filepath.Join(goitDir, objectsDir, header.Name)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("creating object dir %s: %w", filepath.Dir(destPath), err)
		}

		destFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("creating object file %s: %w", destPath, err)
		}

		if _, err := io.Copy(destFile, tarReader); err != nil {
			destFile.Close()
			return fmt.Errorf("writing object file %s: %w", destPath, err)
		}
		destFile.Close()
	}
	return nil
}

func GoitFetch(remoteName string) (*InfoRefsResponse, error) {
	config, err := ReadConfig()
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	remoteSection := "remote \"" + remoteName + "\""
	remoteURL, ok := config[remoteSection]["url"]
	if !ok {
		return nil, fmt.Errorf("remote '%s' not found in config", remoteName)
	}

	infoRefs, err := FetchInfoRefs(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("fetching info/refs from %s: %w", remoteURL, err)
	}

	haves := make([]string, 0)
	remoteRefDir := filepath.Join(goitDir, "refs", "remotes", remoteName)
	entries, _ := os.ReadDir(remoteRefDir)
	for _, entry := range entries {
		if !entry.IsDir() {
			hash, err := local.GetRefHash(filepath.Join("refs", "remotes", remoteName, entry.Name()))
			if err == nil && hash != "" {
				haves = append(haves, hash)
			}
		}
	}

	wants := make([]string, 0, len(infoRefs.Refs))
	for _, hash := range infoRefs.Refs {
		wants = append(wants, hash)
	}

	if len(wants) == 0 {
		fmt.Println("Remote has no branches, nothing to fetch.")
		return infoRefs, nil
	}

	req := GetObjectsRequest{
		Wants: wants,
		Haves: haves,
	}

	tarballStream, err := FetchObjects(remoteURL, req)
	if err != nil {
		return nil, fmt.Errorf("fetching objects from %s: %w", remoteURL, err)
	}

	if err := UnpackObjects(tarballStream); err != nil {
		return nil, fmt.Errorf("unpacking objects: %w", err)
	}

	for ref, hash := range infoRefs.Refs {
		if strings.HasPrefix(ref, "refs/heads/") {
			localRef := strings.Replace(ref, "refs/heads/", "refs/remotes/"+remoteName+"/", 1)
			if err := local.UpdateRef(localRef, hash); err != nil {
				return nil, fmt.Errorf("updating remote ref %s: %w", localRef, err)
			}
		}
	}

	return infoRefs, nil
}

func GoitClone(cloneURL string, directory string) error {
	if directory == "" {
		parsedURL, err := url.Parse(cloneURL)
		if err != nil {
			return fmt.Errorf("invalid clone URL: %w", err)
		}
		base := filepath.Base(parsedURL.Path)
		directory = strings.TrimSuffix(base, ".git")
		if directory == "" || directory == "." || directory == "/" {
			return fmt.Errorf("cannot deduce repository name from URL: %s", cloneURL)
		}
	}

	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		return fmt.Errorf("destination path '%s' already exists and is not an empty directory", directory)
	}

	if err := os.Mkdir(directory, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", directory, err)
	}

	originalWd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(directory); err != nil {
		return fmt.Errorf("changing to directory %s: %w", directory, err)
	}
	defer os.Chdir(originalWd)

	if err := local.InitRepository(".", false); err != nil {
		return fmt.Errorf("initializing repository in %s: %w", directory, err)
	}

	config, err := ReadConfig()
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	remoteSection := "remote \"origin\""
	config[remoteSection] = make(map[string]string)
	config[remoteSection]["url"] = cloneURL
	if err := config.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("Cloning into '%s'...\n", directory)

	infoRefs, err := GoitFetch("origin")
	if err != nil {
		return fmt.Errorf("fetch failed during clone: %w", err)
	}

	if infoRefs.Head == "" {
		fmt.Println("Warning: remote repository is empty.")
		return nil
	}

	defaultBranchRef := ""
	if strings.HasPrefix(infoRefs.Head, "ref: ") {
		defaultBranchRef = strings.TrimPrefix(infoRefs.Head, "ref: ")
	} else {
		return fmt.Errorf("remote HEAD is detached, cannot determine default branch")
	}

	defaultBranchName := strings.TrimPrefix(defaultBranchRef, "refs/heads/")
	if defaultBranchName == "" {
		return fmt.Errorf("could not parse default branch from HEAD: %s", infoRefs.Head)
	}

	defaultBranchHash, ok := infoRefs.Refs[defaultBranchRef]
	if !ok {
		return fmt.Errorf("remote HEAD points to %s but that ref was not found in remote refs", defaultBranchRef)
	}

	if err := local.UpdateRef("refs/heads/"+defaultBranchName, defaultBranchHash); err != nil {
		return fmt.Errorf("failed to create local branch '%s': %w", defaultBranchName, err)
	}

	fmt.Printf("Checking out default branch '%s'\n", defaultBranchName)
	if _, err := local.Checkout(defaultBranchName); err != nil {
		return fmt.Errorf("failed to checkout default branch '%s': %w", defaultBranchName, err)
	}

	return nil
}
