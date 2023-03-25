package cmd

import (
	"errors"
	"fmt"
	"ksd/internal/ksd"
	"os"
	"path/filepath"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Builds stored Kusto functions and tables into scripts suitable for deployment.",
	Args:  cobra.ExactArgs(1),
	Long: heredoc.Doc(`
		Build builds all Kusto file declarations under the current directory.
		To build from a subdirectory, simply pass the path to the directory as an argument.

	    Build does the following:
		- Parses comments that decorate a Kusto function or table declaration as documentation.
		- Transpiles function declarations in user-defined syntax to Stored Function declarations.`),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := os.Getwd()
		if err != nil {
			return err
		}
		if len(args) == 1 {
			root = filepath.Join(root, args[0])
		}

		_, err = os.Stat(root)
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("directory %s does not exist", root)
		}
		if err != nil {
			return err
		}

		outRoot := filepath.Join(root, ".out")
		if err := os.MkdirAll(outRoot, 0755); err != nil {
			return err
		}

		return ksd.Build(root, outRoot)
	},
}

func init() {
}
