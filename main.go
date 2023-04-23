/*
Copyright © 2023 Wei Lim
*/
package main

import (
	"ksd/cmd"
	"os"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
