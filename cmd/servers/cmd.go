package servers

import "github.com/spf13/cobra"

func NewServersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "servers COMMAND",
		Short: "Manage servers",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		newListCommand(),
	)

	return cmd
}
