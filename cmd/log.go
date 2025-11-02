package cmd

import (
	"fmt"
	"souvik606/goit/pkg/goit"
	"strings"

	"github.com/spf13/cobra"
)

const (
	ColorYellow = "\x1b[33m"
	ColorCyan   = "\x1b[36m"
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
			fmt.Printf("%scommit %s%s\n", ColorYellow, entry.Hash, ColorReset)
			fmt.Printf("Author: %s%s%s\n", ColorCyan, entry.Commit.AuthorLine, ColorReset)
			fmt.Printf("Committer: %s%s%s\n", ColorCyan, entry.Commit.CommitterLine, ColorReset)

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
