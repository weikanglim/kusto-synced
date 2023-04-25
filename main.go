/*
Copyright © 2023 Wei Lim
*/
package main

import (
	"os"

	"github.com/weikanglim/ksd/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
