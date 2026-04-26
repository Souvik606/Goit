package local_cmd

import (
	"fmt"
	"os"

	goit "souvik606/goit/pkg/goit/local"

	"github.com/spf13/cobra"
)

var stashCmd = &cobra.Command{
	Use:   "stash",
	Short: "Stash the changes in a dirty working directory away",
	Long:  `Saves your local modifications away and reverts the working directory to match the HEAD commit.`,
	Run: func(cmd *cobra.Command, args []string) {
		verifyRepo()

		msg, err := goit.Stash()
		if err != nil {
			fmt.Printf("Error stashing: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(msg)
	},
}

var stashPopCmd = &cobra.Command{
	Use:   "pop",
	Short: "Apply the changes recorded in the stash to the working tree",
	Run: func(cmd *cobra.Command, args []string) {
		verifyRepo()

		msg, err := goit.StashPop()
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		fmt.Println(msg)
	},
}

func init() {
	stashCmd.AddCommand(stashPopCmd)
	RootCmd.AddCommand(stashCmd)
}

func verifyRepo() {
	if _, err := os.Stat(".goit"); os.IsNotExist(err) {
		if !goit.IsValidBareRepo(".") {
			fmt.Println("fatal: not a goit repository (or any of the parent directories): .goit")
			os.Exit(1)
		}
	}
}
