package local_cmd

import (
	"fmt"
	goit "souvik606/goit/pkg/goit/local"

	"github.com/spf13/cobra"
)

var bare bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Goit repository",
	Long:  `Creates the .goit directory structure or a bare repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := goit.InitRepository(bare)
		if err != nil {
			return err
		}

		if bare {
			fmt.Println("Initialized empty bare Goit repository.")
		} else {
			fmt.Println("Initialized empty Goit repository.")
		}
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&bare, "bare", false, "Create a bare repository")
	RootCmd.AddCommand(initCmd)
}
