package apps

import "github.com/spf13/cobra"

func NewAppsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps COMMAND",
		Short: "Manage apps",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		newListCommand(),
	)

	return cmd
}
