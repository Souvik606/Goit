package local_cmd

import (
	"fmt"
	"os"
	goit "souvik606/goit/pkg/goit/local"

	"github.com/spf13/cobra"
)

var catFileCmd = &cobra.Command{
	Use:   "cat-file <object-hash>",
	Short: "Provide content for repository objects",
	Long:  `Retrieves the content of a Git object (blob, tree, commit) from the database given its SHA-1 hash.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		objectHash := args[0]
		_, content, err := goit.CatFile(objectHash)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("fatal: Not a valid object name %s", objectHash)
			}
			return fmt.Errorf("reading object %s: %w", objectHash, err)
		}

		_, err = os.Stdout.Write(content)
		if err != nil {
			return fmt.Errorf("writing object content to stdout: %w", err)
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(catFileCmd)
}
