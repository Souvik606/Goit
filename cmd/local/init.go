package cmd

import (
	"fmt"
	"os"
	goit "souvik606/goit/pkg/goit/local"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialises a new repository",
	Long:  `Creates the .goit directory structure`,
	Run: func(cmd *cobra.Command, args []string) {
		err := goit.InitRepository()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error initializing repository", err)
			os.Exit(1)
		}
		fmt.Println("Initialized empty goit repository")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
