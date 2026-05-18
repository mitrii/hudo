// Package main is the entrypoint for remote-sudo.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"remote-sudo/cmd"
)

func main() {
	root := &cobra.Command{
		Use:   "remote-sudo",
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
