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

* Goit requires Go 1.25.1 or higher.
* The CLI interface is built using the `cobra` framework.

```bash
# Clone the source code
git clone [https://github.com/Souvik606/goit.git](https://github.com/Souvik606/goit.git)
cd goit

# Build the binary
go build -o goit main.go

# Add to your PATH (optional)
mv goit /usr/local/bin/
```

## Hosting Your Own Goit Server (Acting as GitHub)

Because Goit is a distributed system, it includes a built-in HTTP server that allows you to easily host your own centralized remote repositories—acting exactly like a self-hosted GitHub.

### Step 1: Create a Bare Repository on the Server
A "bare" repository is a special type of repository that doesn't have a working directory. It only stores the version control database, making it perfect for acting as a central hub.

```bash
# Create a folder for your central repositories
mkdir -p /path/to/server-repos/my-project.git
cd /path/to/server-repos/my-project.git

# Initialize it as a bare repository
goit init --bare
```
### Step 2: Start the Goit Server
Run the built-in HTTP server and point it to the root directory where your bare repositories live (the folder containing my-project.git).

```bash
cd /path/to/server-repos
goit serve -p 8080
```

Your Goit server is now live at http://localhost:8080!

### Step 3: Connect from a Client Machine
Now, developers can clone, push, and pull from this central server across the network.

# Clone the repository to a local machine
```bash
goit clone http://localhost:8080/my-project.git client-workspace
cd client-workspace
```

# Make changes and push back to the server
```bash
echo "Hello from the client!" > test.txt
goit add .
goit commit -m "First commit to the server"
goit push origin main
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
