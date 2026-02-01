package remote_cmd

import (
	"fmt"
	"souvik606/goit/cmd/local_cmd"
	goit "souvik606/goit/pkg/goit/remote"

	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch <remote-name>",
	Short: "Download objects and refs from another repository",
	Long: `Fetch branches and/or tags from one or more other repositories,
along with the objects necessary to complete their histories.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remoteName := args[0]

		fmt.Printf("Fetching from %s...\n", remoteName)
		_, err := goit.GoitFetch(remoteName)
		if err != nil {
			return err
		}

		fmt.Println("Fetch complete.")
		return nil
	},
}

func init() {
	local_cmd.RootCmd.AddCommand(fetchCmd)
}
