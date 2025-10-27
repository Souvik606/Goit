package cmd

import (
	"fmt"
	"souvik606/goit/pkg/goit"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [<pathspec>...]",
	Short: "Add file contents to the index",
	Long:  `This command updates the index using the current content found in the working tree, to prepare the content staged for the next commit.If no pathspec is given, all changes in the working directory are staged.`,
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		index := goit.NewIndex()
		if err := index.Load(); err != nil {
			return fmt.Errorf("loading index: %w", err)
		}

		pathsToAdd := args
		if len(args) == 0 {
			pathsToAdd = []string{"."}
		}

		if err := goit.AddPaths(pathsToAdd, index); err != nil {
			saveErr := index.Save()
			if saveErr != nil {
				return fmt.Errorf("saving index after add errors: %w (original add error: %v)", saveErr, err)
			}
			return fmt.Errorf("adding paths: %w", err)
		}

		if err := index.Save(); err != nil {
			return fmt.Errorf("saving index: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
