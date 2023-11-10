/*
Copyright © 2023 Wei Lim
*/
package cmd

import (
	"io"
	"log"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	var debug bool

	root := &cobra.Command{
		Use:          "ksd",
		SilenceUsage: true,
		Short:        "ksd hlpes simplifies and accelerates development for Kusto.",
		Example: heredoc.Doc(`
		# sync files under current directory
		$ ksd sync --endpoint https://<cluster>.kusto.windows.net/<database>

		# sync files under src/kusto directory
		$ ksd sync src/kusto --endpoint https://<cluster>.kusto.windows.net/<database>

		# Sync files in CI (app credential).
		$ ksd sync --endpoint https://<cluster>.kusto.windows.net/<database> --client-id <clientId> --client-secret <clientSecret> --tenantId <tenantId>
		`),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !debug {
				log.SetOutput(io.Discard)
			}
			return nil
		},
	}
	root.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")

	root.AddCommand(NewBuildCommand())
	root.AddCommand(NewSyncCommand())
	root.AddCommand(NewRunCmd())

	return root
}
