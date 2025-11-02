package remote_cmd

import (
	"fmt"
	"os"
	"souvik606/goit/cmd/local_cmd"
	goit "souvik606/goit/pkg/goit/remote"

	"github.com/spf13/cobra"
)

var port string

var serveCmd = &cobra.Command{
	Use:   "serve <base-path>",
	Short: "Start a Goit HTTP server to serve repositories",
	Long: `Starts a multi-repository HTTP server for cloning and fetching.
The <base-path> argument is the root directory containing bare repositories.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		basePath := args[0]

		stat, err := os.Stat(basePath)
		if os.IsNotExist(err) {
			return fmt.Errorf("base path '%s' does not exist", basePath)
		}
		if err != nil {
			return fmt.Errorf("checking base path: %w", err)
		}
		if !stat.IsDir() {
			return fmt.Errorf("base path '%s' is not a directory", basePath)
		}

		if port == "" {
			port = "8080"
		}

		err = goit.Serve(basePath, ":"+port)
		if err != nil {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	},
}

func init() {
	serveCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to listen on")
	local_cmd.RootCmd.AddCommand(serveCmd)
}
