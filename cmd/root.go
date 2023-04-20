/*
Copyright © 2023 Wei Lim
*/
package cmd

import (
	"io"
	"log"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

var debug bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
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
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(syncCmd)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ksd.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")
}
