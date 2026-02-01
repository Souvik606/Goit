package local_cmd

import (
	"fmt"
	"os"
	"path/filepath"

	goit "souvik606/goit/pkg/goit/local"

	"github.com/spf13/cobra"
)

var bare bool

const goitDir = ".goit"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Goit repository",
	Long:  `Creates the .goit directory structure or a bare repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}

		err = goit.InitRepository(wd, bare)
		if err != nil {
			return err
		}

		if bare {
			fmt.Printf("Initialized empty bare Goit repository in %s\n", wd)
		} else {
			fmt.Printf("Initialized empty Goit repository in %s\n", filepath.Join(wd, goitDir))
		}
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&bare, "bare", false, "Create a bare repository")
	RootCmd.AddCommand(initCmd)
}
