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
	"path"
	"path/filepath"
	"souvik606/goit/pkg/goit/local"
	"strings"
)

const goitDir = ".goit"

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
	config, err := local.ReadConfig()
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

	config, err := local.ReadConfig()
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

	fmt.Printf("Checking out default branch '%s'\n", defaultBranchName)

	if _, err := local.Checkout(defaultBranchHash); err != nil {
		return fmt.Errorf("failed to checkout default branch '%s': %w", defaultBranchHash, err)
	}

	if err := local.UpdateRef("refs/heads/"+defaultBranchName, defaultBranchHash); err != nil {
		return fmt.Errorf("failed to create local branch '%s': %w", defaultBranchName, err)
	}

	if err := local.UpdateHead("refs/heads/"+defaultBranchName, ""); err != nil {
		return fmt.Errorf("failed to update HEAD to branch '%s': %w", defaultBranchName, err)
	}

	return nil
}

func GoitPush(remoteName, branchName string) error {
	cfg, err := local.ReadConfig()
	if err != nil {
		return err
	}

	remoteURL, ok := cfg[fmt.Sprintf("remote \"%s\"", remoteName)]["url"]
	if !ok {
		return fmt.Errorf("remote '%s' not found", remoteName)
	}

	localRefPath := "refs/heads/" + branchName
	localHash, err := local.ResolveRef(".goit", localRefPath)
	if err != nil {
		return fmt.Errorf("branch '%s' does not exist locally", branchName)
	}

	infoRefs, err := FetchInfoRefs(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to contact remote: %w", err)
	}

	remoteHash := infoRefs.Refs[localRefPath]
	if remoteHash == "" {
		remoteHash = "0000000000000000000000000000000000000000"
	}

	if localHash == remoteHash {
		fmt.Println("Everything up-to-date")
		return nil
	}

	commitsToSync, err := FindCommitsToSync([]string{localHash}, []string{remoteHash})
	if err != nil {
		return fmt.Errorf("calculating push list: %w", err)
	}

	if len(commitsToSync) == 0 {
		fmt.Println("Everything up-to-date (No new commits found)")
		return nil
	}

	fmt.Printf("Pushing %d commits to %s...\n", len(commitsToSync), remoteURL)

	objectsToPack, err := FindRequiredObjects(commitsToSync)
	if err != nil {
		return fmt.Errorf("preparing objects: %w", err)
	}

	var allHashes []string
	for hash := range commitsToSync {
		allHashes = append(allHashes, hash)
	}
	for hash := range objectsToPack {
		allHashes = append(allHashes, hash)
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for _, hash := range allHashes {
		objPath := filepath.Join(".goit", "objects", hash[:2], hash[2:])
		f, err := os.Open(objPath)
		if err != nil {
			fmt.Printf("Warning: failed to read object %s: %v\n", hash, err)
			continue
		}

		stat, _ := f.Stat()
		hdr := &tar.Header{
			Name: path.Join(hash[:2], hash[2:]),
			Size: stat.Size(),
			Mode: 0644,
		}
		if err := tw.WriteHeader(hdr); err == nil {
			io.Copy(tw, f)
		}
		f.Close()
	}

	tw.Close()
	gw.Close()

	reqURL := fmt.Sprintf("%s/receive-pack?ref=%s&old=%s&new=%s",
		strings.TrimSuffix(remoteURL, "/"),
		localRefPath,
		remoteHash,
		localHash,
	)

	req, err := http.NewRequest("POST", reqURL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-tar")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("push failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server rejected push: %s", string(body))
	}

	trackingRef := "refs/remotes/" + remoteName + "/" + branchName
	local.UpdateRef(trackingRef, localHash)

	fmt.Println("Push successful.")
	return nil
}
