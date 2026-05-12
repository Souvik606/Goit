# Goit: A Distributed Version Control System

Goit is a custom, from-scratch implementation of a Git-like Distributed Version Control System written entirely in Go.

Rather than wrapping existing Git binaries, Goit implements the fundamental systems engineering concepts of version control natively. It features a Directed Acyclic Graph (DAG) state machine, a custom binary index, zlib-compressed object storage, an advanced 3-Way merge algorithm, and a custom HTTP server for distributed network syncing.

## Core Architecture & Systems Design

### 1. The Object Database (Content-Addressable Storage)
* Goit stores repository data as loose objects (blobs, trees, and commits) in the `.goit/objects` directory.
* Every file, directory state, and commit is hashed using SHA-1.
* Objects are tightly compressed using Go's native `compress/zlib` before being written to disk, drastically reducing the local storage footprint.

### 2. The Custom Binary Index (Staging Area)
* The staging area is powered by a highly optimized, custom binary file format.
* The index begins with a strict `GOITIDX` signature and version header.
* It tracks file metadata including permissions (mode), cryptographic hashes, file sizes, and precise modification timestamps (down to the nanosecond) to quickly detect workspace modifications without rehashing files.
* A trailing SHA-1 checksum ensures the binary index file is never corrupted.

### 3. The Merge Engine & Diff Algorithm
Goit natively supports non-linear history and branch merging.
* **Fast-Forward:** Automatically detects if histories are linear and fast-forwards the pointer.
* **3-Way Merge:** If histories diverge, Goit finds the Lowest Common Ancestor (Merge Base) using a Breadth-First Search across the commit DAG.
* **Diff3:** It utilizes a custom Longest Common Subsequence (LCS) algorithm to compute line-by-line diffs (`computeLCSDiff`), and a Diff3 engine to safely auto-merge changes or inject conflict markers (`<<<<<<<`, `=======`, `>>>>>>>`) when lines collide.

### 4. Distributed HTTP Networking
Goit is capable of distributed synchronization across different machines.
* **Custom HTTP Server:** Includes a built-in server (`goit serve`) that acts as a remote host for bare repositories.
* **Efficient Payloads:** During push, pull, and fetch, the system determines exactly which commits the client lacks, packages the required loose objects into an in-memory `tar` archive, compresses it using `gzip`, and streams it over the network to minimize bandwidth.
* **Smart Endpoints:** The server utilizes smart endpoints like `/info/refs` (for discovery), `/get-objects` (for fetching), and `/receive-pack` (for pushing).

## Installation & Setup

You can install Goit either by building it from the source code or by downloading a pre-built standalone binary.

### Option 1: Build from Source (Requires Go)
Goit requires Go 1.25.1 or higher. The CLI interface is built using the `cobra` framework.

```bash
# Clone the source code
git clone [https://github.com/Souvik606/goit.git](https://github.com/Souvik606/goit.git)
cd goit

# Install the binary directly to your GOPATH
go install .

# Alternatively, you can build the binary manually
go build -o goit main.go

# Add to your PATH (if not using 'go install')
mv goit /usr/local/bin/
```

### Option 2: Using Pre-built Binaries (No Go Required)
If you do not have Go installed on your machine, you can download the standalone executables directly from GitHub Releases.

Navigate to the Releases page of this repository.

Download the binary matching your operating system and architecture (e.g., ```goit-windows-amd64.exe```, ```goit-linux-amd64```, ```goit-darwin-amd64```).

Rename the downloaded file to goit (or goit.exe on Windows).

Move the file into a directory that is in your system's PATH (e.g., /usr/local/bin for Linux/Mac, or add the folder to your Windows Environment Variables).

Open your terminal and run goit to verify the installation.

## Getting Help
Once installed, you can learn more about the tool and explore available commands by simply running:

```
goit
# or
goit help
```

## Complete Guide: Hosting and Collaborating with Goit

Because Goit is a true Distributed Version Control System, it includes a built-in HTTP server to handle network synchronization. This guide will walk you through setting up a central server and demonstrating how two different developers (User A and User B) can collaborate on the same repository.

### Phase 1: Setting up the Central Server
First, we need to create a "bare" repository to act as our central hub (similar to a GitHub repository) and start the Goit HTTP server to listen for network requests.

```bash
# 1. Create a directory to hold your server repositories
mkdir -p /path/to/server-repos/test-repo
cd /path/to/server-repos/test-repo

# 2. Initialize it as a bare repository (no working directory)
goit init --bare

# 3. Start the Goit HTTP server in the parent directory
cd /path/to/server-repos
goit serve -p 8080 .
```

The Goit server is now live at http://localhost:8080 and listening for connections.

### Phase 2: User A Initializes and Pushes Code
User A wants to start working on the project, link it to the central server, and push the very first commit.

```bash
# 1. User A creates a local workspace
mkdir ~/user-a-workspace
cd ~/user-a-workspace

# 2. Initialize a standard Goit repository
goit init

# 3. Create some initial files
echo "Hello from User A" > main.txt
goit add .
goit commit -m "Initial commit by User A"

# 4. Link the local repository to the central server
goit remote add origin http://localhost:8080/test-repo

# 5. Push the code to the central server
goit push origin main
```

### Phase 3: User B Clones and Contributes
User B joins the team and needs to pull down the existing code, make their own changes, and push them back.

```bash
# 1. User B clones the repository directly from the server
goit clone http://localhost:8080/test-repo ~/user-b-workspace
cd ~/user-b-workspace

# 2. User B makes a new file and commits the changes
echo "User B's new feature" > feature.txt
goit add .
goit commit -m "Added new feature file"

# 3. User B pushes their changes back to the server
# (The remote 'origin' is automatically configured during clone)
goit push origin main
```

### Phase 4: User A Synchronizes (Pulls Changes)
User A now needs to update their local workspace to include the new feature built by User B.

```bash
# 1. User A returns to their workspace
cd ~/user-a-workspace

# 2. User A pulls the latest changes from the server
# This will fetch the new loose objects and trigger a fast-forward or 3-way merge
goit pull origin main

# 3. User A can verify the new files exist
cat feature.txt
```

## Command Reference

### Local Repository Commands

* **`goit init [--bare]`**
  Initializes a new `.goit` directory structure. Use `--bare` to create a headless server repository.
* **`goit add <pathspec>`**
  Hashes files, compresses them, and stages them in the binary index. Parses `.goitignore` dynamically.
* **`goit commit -m "<message>"`**
  Takes a snapshot of the current index, writes a tree object, and generates a commit linked to the DAG history.
* **`goit status`**
  Compares the Workspace, Index, and HEAD tree to display Staged, Unstaged, and Untracked files.
* **`goit branch [<name>]`**
  Lists all branches or creates a new pointer to the current HEAD.
* **`goit checkout <target>`**
  Moves the HEAD pointer, cleans the workspace, and extracts the target tree from the object database.
* **`goit merge <branch>`**
  Performs a Fast-Forward or 3-Way merge of the target branch into the current HEAD.
* **`goit reset [--soft | --mixed | --hard] <commit>`**
  Manipulates the HEAD, index, and workspace to match a previous timeline state.
* **`goit rm [--cached] [-r] <file>`**
  Removes files from the index and physically deletes them from the hard drive (unless `--cached` is used).
* **`goit stash` / `goit stash pop`**
  Saves dirty working directory states to a local stack and restores them cleanly.
* **`goit diff`**
  Shows exact line insertions and deletions between the workspace and the index.
* **`goit log`**
  Traverses the DAG backward from HEAD to display commit history.

### Network / Remote Commands

* **`goit serve -p <port> <directory>`**
  Spins up the Goit HTTP server to host bare repositories for remote access.
* **`goit remote add <name> <url>`**
  Saves remote URL configurations to the local `.goit/config` file.
* **`goit clone <url> [<dir>]`**
  Initializes a new repo, configures the remote, fetches objects, and checks out the default branch.
* **`goit push <remote> <branch>`**
  Calculates missing commits, packages objects into a `tar.gz` stream, and pushes them to the server.
* **`goit fetch <remote>`**
  Downloads required loose objects and updates `refs/remotes/<name>` pointers.
* **`goit pull [<remote>] [<branch>]`**
  Fetches the remote objects and immediately triggers a merge into the current local branch.

## Testing & Reliability

The engine is heavily verified through a suite of automated integration tests located in the `test/` directory, achieving high behavioral parity with actual Git workflows. The test suite explicitly validates complex systems such as:

* DAG traversal and detached HEAD states.
* 3-Way merge conflict generation and resolution.
* Index corruption and checksum validation.
* End-to-end network cloning and pushing via local HTTP test servers.
