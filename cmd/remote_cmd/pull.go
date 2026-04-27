package remote_cmd

import (
	"fmt"
	"strings"

	"souvik606/goit/cmd/local_cmd"
	goit_local "souvik606/goit/pkg/goit/local"
	goit_remote "souvik606/goit/pkg/goit/remote"

	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull [<remote>] [<branch>]",
	Short: "Fetch from and integrate with another repository",
	Long:  `Incorporates changes from a remote repository into the current branch. This is shorthand for 'goit fetch' followed by 'goit merge'.`,
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		remoteName := "origin"
		headRef, err := goit_local.GetHeadRef()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		branchName := "main"
		if strings.HasPrefix(headRef, "refs/heads/") {
			branchName = strings.TrimPrefix(headRef, "refs/heads/")
		}

		if len(args) >= 1 {
			remoteName = args[0]
		}
		if len(args) >= 2 {
			branchName = args[1]
		}

		fmt.Printf("Fetching from %s...\n", remoteName)
		_, err = goit_remote.GoitFetch(remoteName)
		if err != nil {
			return fmt.Errorf("fetch failed: %w", err)
		}
		fmt.Println("Fetch complete.")

		targetRef := fmt.Sprintf("refs/remotes/%s/%s", remoteName, branchName)
		displayRef := fmt.Sprintf("%s/%s", remoteName, branchName)

		fmt.Printf("Merging %s into local branch...\n", displayRef)

		requires3Way, mergeBaseHash, err := goit_local.Merge(targetRef)
		if err != nil {
			return err
		}

		if requires3Way {
			headHash, err := goit_local.GetHeadCommitHash()
			if err != nil {
				return fmt.Errorf("failed to get HEAD hash: %w", err)
			}

			targetHash, _, err := goit_local.ResolveTarget(targetRef)
			if err != nil {
				return fmt.Errorf("failed to resolve target hash: %w", err)
			}

			err = goit_local.Execute3WayMerge(mergeBaseHash, headHash, targetHash, displayRef)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	local_cmd.RootCmd.AddCommand(pullCmd)
}
