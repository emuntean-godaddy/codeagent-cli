package cmd

import (
	"github.com/emuntean-godaddy/codeagent-cli/internal/errutil"
	"github.com/spf13/cobra"
)

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "codeagent",
		Short: "Manage project devcontainers",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return errutil.UserError("unknown command")
		},
	}

	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newDestroyCmd())
	cmd.AddCommand(newDoctorCmd())

	return cmd
}
