package local_cmd

import (
	goit "souvik606/goit/pkg/goit/local"

	"github.com/spf13/cobra"
)

var rmCached bool
var rmRecursive bool

var rmCmd = &cobra.Command{
	Use:   "rm <file>...",
	Short: "Remove files from the working tree and from the index",
	Long:  `Remove files from the index, or from the working tree and the index. goit rm will not remove a file from just your working directory.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return goit.Rm(args, rmCached, rmRecursive)
	},
}

func init() {
	rmCmd.Flags().BoolVar(&rmCached, "cached", false, "only remove from the index")
	rmCmd.Flags().BoolVarP(&rmRecursive, "r", "r", false, "allow recursive removal")

	RootCmd.AddCommand(rmCmd)
}
