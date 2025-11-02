package cmd

import (
	"fmt"
	goit "souvik606/goit/pkg/goit/local"

	"github.com/spf13/cobra"
)

var checkoutCmd = &cobra.Command{
	Use:   "checkout <branch-or-commit>",
	Short: "Switch branches or restore working tree files",
	Long: `Switches to a different branch or checks out a specific commit,
updating the working directory and index to match.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		message, err := goit.Checkout(target)
		if err != nil {
			return err
		}

		fmt.Println(message)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
}
