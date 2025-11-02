package cmd

import (
	"fmt"
	"os"
	goit "souvik606/goit/pkg/goit/local"

	"github.com/spf13/cobra"
)

const (
	ColorGreen = "\x1b[32m"
	ColorReset = "\x1b[0m"
)

var branchCmd = &cobra.Command{
	Use:   "branch [<branch-name>]",
	Short: "List, create, or delete branches",
	Long: `With no arguments, list all existing branches.
If <branch-name> is provided, create a new branch pointing to the current HEAD commit.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return listBranches()
		} else {
			return createBranch(args[0])
		}
	},
}

func listBranches() error {
	branches, activeBranch, err := goit.ListBranches()
	if err != nil {
		return err
	}

	for _, branch := range branches {
		if branch == activeBranch {
			fmt.Printf("* %s%s%s\n", ColorGreen, branch, ColorReset)
		} else {
			fmt.Printf("  %s\n", branch)
		}
	}
	return nil
}

func createBranch(name string) error {
	err := goit.CreateBranch(name)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Branch '%s' created.\n", name)
	return nil
}

func init() {
	rootCmd.AddCommand(branchCmd)
}
