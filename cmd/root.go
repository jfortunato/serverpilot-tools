package cmd

import (
	"fmt"
	"github.com/jfortunato/serverpilot-tools/cmd/apps"
	"github.com/jfortunato/serverpilot-tools/cmd/servers"
	"github.com/spf13/cobra"
	"os"
)

type VersionDetails struct {
	Version string
	Commit  string
	Date    string
}

var rootCmd = &cobra.Command{
	Use:   "serverpilot-tools",
	Short: "A collection of tools for ServerPilot.io",
	Long:  `A collection of tools for ServerPilot.io`,
}

func Execute(v VersionDetails) {
	rootCmd.Version = v.Version

	rootCmd.AddCommand(
		apps.NewAppsCommand(),
		servers.NewServersCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
