package cmd

import (
	"fmt"
	"souvik606/goit/pkg/goit"
	"strings"

	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show commit logs",
	Long:  `Displays the commit history of the current branch.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		history, err := goit.Log()
		if err != nil {
			return err
		}

		if len(history) == 0 {
			fmt.Println("No commits yet")
			return nil
		}

		for _, entry := range history {
			fmt.Printf("commit %s\n", entry.Hash)
			fmt.Printf("Author: %s\n", entry.Commit.AuthorLine)
			fmt.Printf("Committer: %s\n", entry.Commit.CommitterLine)

			fmt.Println()

			for _, line := range strings.Split(entry.Commit.Message, "\n") {
				fmt.Printf("    %s\n", line)
			}
			fmt.Println()
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
