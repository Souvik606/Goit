package cmd

import (
	"fmt"
	goit "souvik606/goit/pkg/goit/local"
	"strings"

	"github.com/spf13/cobra"
)

var commitMsg string

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Record changes to the repository",
	Long:  `Creates a new commit containing the current contents of the index and the given log message.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if commitMsg == "" {
			return fmt.Errorf("aborting commit due to empty commit message (use -m flag)")
		}

		newCommitHash, refUpdated, err := goit.Commit(commitMsg)
		if err != nil {
			return err
		}

		branchName := strings.TrimPrefix(refUpdated, "refs/heads/")
		firstLine := strings.Split(commitMsg, "\n")[0]

		fmt.Printf("[%s %s] %s\n", branchName, newCommitHash[:7], firstLine)

		return nil
	},
}

func init() {
	commitCmd.Flags().StringVarP(&commitMsg, "message", "m", "", "Commit message(required)")
	commitCmd.MarkFlagRequired("message")
	rootCmd.AddCommand(commitCmd)
}
