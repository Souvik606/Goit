package local_cmd

import (
	"fmt"
	"strings"

	goit "souvik606/goit/pkg/goit/local"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config <key> <value>",
	Short: "Get and set repository options",
	Long:  `Set configuration values like author identity. Example: goit config user.name "name"`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyArg := args[0]
		valueArg := args[1]

		parts := strings.SplitN(keyArg, ".", 2)
		if len(parts) != 2 {
			return fmt.Errorf("key does not contain a section: %s (expected format section.key)", keyArg)
		}
		section, key := parts[0], parts[1]

		config, err := goit.ReadConfig()
		if err != nil {
			return fmt.Errorf("reading config: %w", err)
		}

		if config[section] == nil {
			config[section] = make(map[string]string)
		}

		config[section][key] = valueArg

		if err := config.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(configCmd)
}
