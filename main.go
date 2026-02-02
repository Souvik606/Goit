package main

import (
	cmd "souvik606/goit/cmd/local_cmd"
	_ "souvik606/goit/cmd/remote_cmd"
)

func main() {
	cmd.Execute()
}
