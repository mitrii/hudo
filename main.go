// Package main is the entrypoint for hudo.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"hudo/cmd"
)

const longHelp = `hudo — privilege escalation for autonomous agents.

Instead of granting agents unrestricted root access, hudo requires a human
to approve each privileged command via a one-time PIN sent to a webhook.

Workflow:
  1. hudo request <command>          generate PIN, notify human, print short ID
  2. human reads PIN and gives it to the agent
  3. hudo exec <command> --pin <PIN> verify PIN and run command as root

Aliases:
  request → r
  exec    → e

Examples:
  hudo r "systemctl restart nginx"
  hudo e "systemctl restart nginx" --pin 847291

Configuration: /etc/hudo/config.yaml (owner root:root, mode 0600)`

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
