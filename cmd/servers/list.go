package servers

import (
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"github.com/jfortunato/serverpilot-tools/internal/servers"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"text/tabwriter"
)

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [OPTIONS]",
		Aliases: []string{"ls"},
		Short:   "List servers",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := log.New(io.Discard, "", 0)

			c := serverpilot.NewClient(logger, args[0], args[1])

			s, err := servers.GetServers(c)
			if err != nil {
				return fmt.Errorf("error while getting servers: %w", err)
			}

			printServers(s)

			return nil
		},
	}

	return cmd
}

func printServers(servers []serverpilot.Server) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tIP\tCREATED\t")
	for _, server := range servers {
		fmt.Fprintln(w, server.Id+"\t"+server.Name+"\t"+server.Ipaddress+"\t"+server.Datecreated.String()+"\t")
	}
	w.Flush()
}
