package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/weikanglim/ksd/internal/ksd"
)

func NewRunCmd() *cobra.Command {
	var script string

	var runCmd = &cobra.Command{
		Use:   "run <file>",
		Short: "Runs a script file.",
		Args:  cobra.ExactArgs(1),
		Long: heredoc.Doc(`
			Run executes the script file against a Kusto database.
			`),
		Example: heredoc.Doc(`
			# Run a script file
			$ ksd run ./script.ksl
			`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if endpoint == "" {
				return errors.New("missing `--endpoint`. Set this to a Azure Data Explorer database endpoint, i.e. https://samples.kusto.windows.net/MyDatabase")
			}

			file := args[0]
			_, err := os.Stat(file)
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("script file %s does not exist", file)
			}
			if err != nil {
				return fmt.Errorf("reading %s: %w", file, err)
			}

			credOptions, err := GetCredentialOptionsFromFlags()
			if err != nil {
				return err
			}

			return ksd.Run(file, endpoint, credOptions, http.DefaultClient)
		},
	}

	runCmd.Flags().StringVar(&script, "script", "", "The script file to run.")

	// Connection flags
	runCmd.Flags().StringVar(&endpoint, "endpoint", "", "The endpoint to the Azure Data Explorer database")
	runCmd.Flags().StringVar(&clientId, "client-id", "", "The ID of the application to authenticate with")
	runCmd.Flags().StringVar(&clientSecret, "client-secret", "", "The secret of the application to authenticate with")
	runCmd.Flags().StringVar(&tenantId, "tenant-id", "", "The tenant ID of the application to authenticate with")

	return runCmd
}
