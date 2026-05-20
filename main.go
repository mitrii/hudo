// Package main is the entrypoint for hudo.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"hudo/cmd"
)

const longHelp = `Usage:
  hudo request <command>              request PIN to run command as root (alias: r)
  hudo exec <command> --pin <PIN>     execute command after PIN approval (alias: e)

Prints 8-char request ID on stdout. Use it to match the PIN notification.

Config: /etc/hudo/config.yaml`

func main() {
	root := &cobra.Command{
		Use:   "hudo",
		Short: "Privilege escalation for autonomous agents",
		Long:  longHelp,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
			}

			return cmd.Help()
		},
	}

	root.AddCommand(cmd.RequestCmd)
	root.AddCommand(cmd.ExecCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
