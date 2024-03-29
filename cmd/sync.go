package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/weikanglim/ksd/internal/ksd"
)

func NewSyncCommand() *cobra.Command {
	var fromOut string
	var syncCmd = &cobra.Command{
		Use:   "sync <directory>",
		Short: "Syncs Kusto function and table declarations to a targeted Azure Data Explorer database",
		Args:  cobra.MaximumNArgs(1),
		Long: heredoc.Doc(`
		sync will automatically call 'ksd build' to ensure that all files are built into command scripts.
		To skip this behavior, pass the '--from-out' flag specifying the output directory that is already built.

		The command scripts, located in the 'kout' directory (which contain Kusto Management Commands) are loaded and executed against the target Kusto database.
		Thus, sync ends up syncing functions and tables declaration stored locally to the database.`),
		Example: heredoc.Doc(`
		# Sync either using 'az' login credentials, or an interactive login
		$ ksd sync --endpoint https://<cluster>.kusto.windows.net/<database>

		# Sync using aad app credentials. Recommended for CI workflows.
		$ ksd sync --endpoint https://<cluster>.kusto.windows.net/<database> --client-id <clientId> --client-secret <secretId> --tenantId <tenantId>

		# Sync using GitHub OIDC credentials. Recommended for CI workflows.
		$ ksd sync --endpoint https://<cluster>.kusto.windows.net/<database> --client-id <clientId> --credential-provider github --tenantId <tenantId>
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting cwd: %w", err)
			}

			if len(args) > 0 {
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

			if endpoint == "" {
				return errors.New("missing `--endpoint`. Set this to a Azure Data Explorer database endpoint, i.e. https://samples.kusto.windows.net/MyDatabase")
			}

			credOptions, err := GetCredentialOptionsFromFlags()
			if err != nil {
				return err
			}

			var outRoot string
			if fromOut != "" {
				// from-out specified, skip build
				if filepath.IsAbs(fromOut) {
					outRoot = filepath.Clean(fromOut)
				} else {
					outRoot = filepath.Join(root, fromOut)
				}

				if !strings.HasSuffix(outRoot, ksd.OutDir) {
					return fmt.Errorf(
						"%s is an invalid out directory path. out directories are expected to be named '%s'",
						fromOut,
						ksd.OutDir)
				}
			} else {
				// default mode, build to out folder
				outRoot = filepath.Join(root, ksd.OutDir)

				if err := os.MkdirAll(outRoot, 0755); err != nil {
					return err
				}

				fmt.Println("Building files...")
				err = ksd.Build(root, outRoot)
				if err != nil {
					return err
				}
			}

			fmt.Println("Syncing files...")
			return ksd.Sync(
				outRoot,
				endpoint,
				credOptions,
				http.DefaultClient)
		},
	}
	syncCmd.Flags().StringVar(&fromOut, "from-out", "", "The output directory that contains command files to sync.")
	// Connection flags
	syncCmd.Flags().StringVar(&endpoint, "endpoint", "", "The endpoint to the Azure Data Explorer database")
	syncCmd.Flags().StringVar(&clientId, "client-id", "", "The ID of the application to authenticate with")
	syncCmd.Flags().StringVar(&clientSecret, "client-secret", "", "The secret of the application to authenticate with")
	syncCmd.Flags().StringVar(&tenantId, "tenant-id", "", "The tenant ID of the application to authenticate with")
	syncCmd.Flags().StringVar(&credentialProvider, "credential-provider", "", "The credential provider to use instead of client-secret. Allowed values: github")

	return syncCmd
}
