package cmd

import (
	"fmt"
	"sort"
	goit "souvik606/goit/pkg/goit/local"
	"strings"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the working tree status",
	Long: `Displays paths that have differences between the index file and the current HEAD commit, 
paths that have differences between the working tree and the index file, 
and paths in the working tree that are not tracked.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		summary, err := goit.GetStatus()
		if err != nil {
			return err
		}

		currentRef, err := goit.GetHeadRef()
		if err != nil {
			return err
		}

		if strings.HasPrefix(currentRef, "refs/heads/") {
			fmt.Printf("On branch %s\n", strings.TrimPrefix(currentRef, "refs/heads/"))
		} else {
			fmt.Printf("HEAD detached at %s\n", currentRef[:7])
		}

		if len(summary.Staged) == 0 && len(summary.Unstaged) == 0 && len(summary.Untracked) == 0 {
			fmt.Println("nothing to commit, working tree clean")
			return nil
		}

		printStatusSection("Changes to be committed:", summary.Staged, "green")
		printStatusSection("Changes not staged for commit:", summary.Unstaged, "red")

		if len(summary.Untracked) > 0 {
			fmt.Println("\nUntracked files:")
			fmt.Println("  (use \"goit add <file>...\" to include in what will be committed)")
			sort.Strings(summary.Untracked)
			for _, path := range summary.Untracked {
				fmt.Printf("\t\x1b[36m%s\x1b[0m\n", path)
			}
		}

		return nil
	},
}

func printStatusSection(title string, changes map[string]goit.StatusChangeType, color string) {
	if len(changes) == 0 {
		return
	}

	fmt.Printf("\n%s\n", title)

	colorCode := "\x1b[32m"
	if color == "red" {
		colorCode = "\x1b[31m"
	}
	resetCode := "\x1b[0m"

	paths := make([]string, 0, len(changes))
	for path := range changes {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		changeType := changes[path]
		switch changeType {
		case goit.ChangeStagedNew:
			fmt.Printf("\t%snew file:   %s%s\n", colorCode, path, resetCode)
		case goit.ChangeStagedModified:
			fmt.Printf("\t%smodified:   %s%s\n", colorCode, path, resetCode)
		case goit.ChangeStagedDeleted:
			fmt.Printf("\t%sdeleted:    %s%s\n", colorCode, path, resetCode)
		case goit.ChangeModified:
			fmt.Printf("\t%smodified:   %s%s\n", colorCode, path, resetCode)
		case goit.ChangeDeleted:
			fmt.Printf("\t%sdeleted:    %s%s\n", colorCode, path, resetCode)
		}
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
