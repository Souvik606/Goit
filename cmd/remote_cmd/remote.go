package remote_cmd

import (
	"fmt"
	"souvik606/goit/cmd/local_cmd"
	goit "souvik606/goit/pkg/goit/remote"

	"github.com/spf13/cobra"
)

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage set of tracked repositories",
	Long:  `Manage the set of repositories ("remotes") whose branches you track.`,
}

var remoteAddCmd = &cobra.Command{
	Use:   "add <name> <url>",
	Short: "Add a new remote",
	Long:  `Adds a remote named <name> for the repository at <url>.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		url := args[1]

		config, err := goit.ReadConfig()
		if err != nil {
			return fmt.Errorf("reading config: %w", err)
		}

		sectionName := fmt.Sprintf("remote \"%s\"", name)
		if _, ok := config[sectionName]; ok {
			return fmt.Errorf("remote %s already exists", name)
		}

		if config[sectionName] == nil {
			config[sectionName] = make(map[string]string)
		}

		config[sectionName]["url"] = url

		if err := config.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		return nil
	},
}

func init() {
	remoteCmd.AddCommand(remoteAddCmd)
	local_cmd.RootCmd.AddCommand(remoteCmd)
}
