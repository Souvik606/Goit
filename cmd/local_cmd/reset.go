package local_cmd

import (
	"fmt"
	goit "souvik606/goit/pkg/goit/local"

	"github.com/spf13/cobra"
)

var resetSoft bool
var resetMixed bool
var resetHard bool

var resetCmd = &cobra.Command{
	Use:   "reset <commit-or-branch>",
	Short: "Reset current HEAD to the specified state",
	Long:  `Resets the current branch head to <commit> and possibly updates the index and the working tree depending on the mode (--soft, --mixed, --hard).`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		modeCount := 0
		mode := "mixed"

		if resetSoft {
			mode = "soft"
			modeCount++
		}
		if resetMixed {
			mode = "mixed"
			modeCount++
		}
		if resetHard {
			mode = "hard"
			modeCount++
		}

		if modeCount > 1 {
			return fmt.Errorf("fatal: Cannot do a reset with multiple modes simultaneously")
		}

		err := goit.Reset(target, mode)
		if err != nil {
			return err
		}

		if mode == "hard" {
			targetHash, _, _ := goit.ResolveTarget(target)
			fmt.Printf("HEAD is now at %s\n", targetHash[:7])
		}

		return nil
	},
}

func init() {
	resetCmd.Flags().BoolVar(&resetSoft, "soft", false, "Does not touch the index file or the working tree at all")
	resetCmd.Flags().BoolVar(&resetMixed, "mixed", false, "Resets the index but not the working tree (default)")
	resetCmd.Flags().BoolVar(&resetHard, "hard", false, "Resets the index and working tree. Discards all tracked changes")
	RootCmd.AddCommand(resetCmd)
}
