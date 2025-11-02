package cmd

import (
	"fmt"
	"souvik606/goit/pkg/goit"

	"github.com/spf13/cobra"
)

var writeObject bool

var hashObjectCmd = &cobra.Command{
	Use:   "hash-object [-w] <file>",
	Short: "Compute object ID and optionally create an object from a file",
	Long:  `Reads the content of <file>, computes its SHA-1 hash formatted as a Git blob object, and optionally writes it to the object database.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		hash, err := goit.HashObject(filePath, writeObject, "blob")
		if err != nil {
			return fmt.Errorf("hashing object %s: %w", filePath, err)
		}
		fmt.Println(hash)
		return nil
	},
}

func init() {
	hashObjectCmd.Flags().BoolVarP(&writeObject, "write", "w", false, "Actually write the object into the database")
	rootCmd.AddCommand(hashObjectCmd)
}
