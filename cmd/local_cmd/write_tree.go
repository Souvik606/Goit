package local_cmd

import (
	"fmt"
	goit "souvik606/goit/pkg/goit/local"

	"github.com/spf13/cobra"
)

var writeTreeCmd = &cobra.Command{
	Use:   "write-tree",
	Short: "Create a tree object from the current index",
	Long:  `Reads the current staging area (index) and writes a tree object representing that state to the object database. Outputs the hash of the root tree object.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		rootTreeHash, err := goit.WriteTree()
		if err != nil {
			return fmt.Errorf("writing tree object: %w", err)
		}
		fmt.Println(rootTreeHash)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(writeTreeCmd)
}
