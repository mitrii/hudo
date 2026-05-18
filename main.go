// Package main is the entrypoint for hudo.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"hudo/cmd"
)

func main() {
	root := &cobra.Command{
		Use:   "hudo",
		Short: "Privilege escalation for autonomous agents",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	root.AddCommand(cmd.RequestCmd)
	root.AddCommand(cmd.ExecCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
