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
		Short:        "A tool that simplifies and accelerates development for Kusto.",
		Long: heredoc.Doc(`
		Kusto Synced (ksd) is a tool that simplifies and accelerates development for Kusto.
			
		- Store commonly used Kusto functions and tables in source control. Deploy the changes using a single command locally or on CI: `) + "`ksd sync`" +
			heredoc.Doc(`
		- Share reusable functions across teams. Functions are organized in the cluster database using the filesystem directory structure, with first-class support for adding documentation.
		- Write functions and test them in Azure Data Explorer. Once you're happy, store it in a file. ksd automatically transpiles your User-Defined function declarations to Stored Function declarations to be saved in the database.
		`),
		Example: heredoc.Doc(`
		# sync files under current directory
		$ ksd sync --cluster <cluster> --database <database> 

		# sync files under src/kusto directory
		$ ksd sync src/kusto --cluster <cluster> --database <database> 
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

	return root
}
