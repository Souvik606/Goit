package remote_cmd

import (
	"fmt"
	"souvik606/goit/cmd/local_cmd"
	goit "souvik606/goit/pkg/goit/remote"

	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <url> [<directory>]",
	Short: "Clone a repository into a new directory",
	Long:  `Clones a repository from <url> into a new directory. If <directory> is not specified, it is derived from the URL.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cloneURL := args[0]
		directory := ""
		if len(args) == 2 {
			directory = args[1]
		}

		err := goit.GoitClone(cloneURL, directory)
		if err != nil {
			return err
		}

		fmt.Println("Clone successful.")
		return nil
	},
}

func init() {
	local_cmd.RootCmd.AddCommand(cloneCmd)
}
