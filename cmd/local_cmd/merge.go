package local_cmd

import (
	"fmt"
	goit "souvik606/goit/pkg/goit/local"

	"github.com/spf13/cobra"
)

var mergeCmd = &cobra.Command{
	Use:   "merge <commit-or-branch>",
	Short: "Join two or more development histories together",
	Long:  `Incorporates changes from the named commit/branch into the current branch.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		requires3Way, mergeBaseHash, err := goit.Merge(target)
		if err != nil {
			return err
		}

		if requires3Way {
			headHash, err := goit.GetHeadCommitHash()
			if err != nil {
				return fmt.Errorf("failed to get HEAD hash: %w", err)
			}

			targetHash, _, err := goit.ResolveTarget(target)
			if err != nil {
				return fmt.Errorf("failed to resolve target hash: %w", err)
			}

			err = goit.Execute3WayMerge(mergeBaseHash, headHash, targetHash, target)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(mergeCmd)
}
