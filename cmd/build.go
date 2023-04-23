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

func NewBuildCommand() *cobra.Command {
	var buildCmd = &cobra.Command{
		Use:   "build <directory>",
		Short: "Builds stored Kusto functions and tables into command files suitable for deployment.",
		Args:  cobra.MaximumNArgs(1),
		Long: heredoc.Doc(`
			Build transpiles all Kusto declarative file declarations under the current directory,
			into command files under the 'kout' directory relative to the current directory.
	
			To specify a subdirectory, simply pass the <directory> as an argument.
	
			Build does the following:
			- Parses comments that decorate a Kusto function or table declaration as documentation.
			- Transpiles table and function declarations in user-defined syntax to command syntax that creates or alters the functions.`),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			if len(args) == 1 {
				if filepath.IsAbs(args[0]) {
					root = args[0]
				} else {
					root = filepath.Join(root, args[0])
				}
			}

			_, err = os.Stat(root)
			if errors.Is(err, os.ErrNotExist) {
				displayDir := root
				if len(args) > 0 {
					displayDir = args[0]
				}
				return fmt.Errorf("directory %s does not exist", displayDir)
			}
			if err != nil {
				return err
			}

			outRoot := filepath.Join(root, ksd.OutDir)
			if err := os.MkdirAll(outRoot, 0755); err != nil {
				return err
			}

			return ksd.Build(root, outRoot)
		},
	}

	return buildCmd
}
