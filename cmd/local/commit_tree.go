package cmd

import (
	"fmt"
	goit "souvik606/goit/pkg/goit/local"

	"strings"

	"github.com/spf13/cobra"
)

var parentHashes []string
var commitMessage string

var commitTreeCmd = &cobra.Command{
	Use:   "commit-tree <tree-hash>",
	Short: "Create a new commit object",
	Long: `Creates a commit object using the specified tree object hash and parent commit hashes.
Outputs the hash of the new commit object. The commit message is taken from the -m flag or read from stdin.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		treeHash := args[0]
		message := commitMessage

		if message == "" {
			var readErr error
			message, readErr = goit.ReadMessageFromStdin()
			if readErr != nil {
				return readErr
			}
		} else {
			if !strings.HasSuffix(message, "\n") {
				message += "\n"
			}
		}

		commitHash, err := goit.CommitTree(treeHash, parentHashes, message)
		if err != nil {
			return fmt.Errorf("creating commit object: %w", err)
		}
		fmt.Println(commitHash)
		return nil
	},
}

func init() {
	commitTreeCmd.Flags().StringSliceVarP(&parentHashes, "parent", "p", nil, "Specify parent commit object hash (can be used multiple times)")
	commitTreeCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "Provide commit message")
	rootCmd.AddCommand(commitTreeCmd)
}
