package local_cmd

import (
	"fmt"
	"os"

	goit "souvik606/goit/pkg/goit/local"

	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show changes between commits, commit and working tree, etc",
	Long:  `Show changes between the working tree and the index (staging area).`,
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(".goit"); os.IsNotExist(err) {
			if !goit.IsValidBareRepo(".") {
				fmt.Println("fatal: not a goit repository (or any of the parent directories): .goit")
				os.Exit(1)
			}
		}

		err := goit.DiffWorkspaceIndex()
		if err != nil {
			fmt.Printf("Error generating diff: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(diffCmd)
}
