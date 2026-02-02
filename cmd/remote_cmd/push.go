package remote_cmd

import (
	"souvik606/goit/cmd/local_cmd"
	goit "souvik606/goit/pkg/goit/remote"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <remote> <branch>",
	Short: "Update remote refs along with associated objects",
	Long:  `Uploads the specified branch and its history to the remote repository.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		remoteName := args[0]
		branchName := args[1]

		err := goit.GoitPush(remoteName, branchName)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	local_cmd.RootCmd.AddCommand(pushCmd)
}
