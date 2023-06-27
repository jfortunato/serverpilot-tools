package apps

import (
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/dns"
	"github.com/jfortunato/serverpilot-tools/internal/filter"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"github.com/jfortunato/serverpilot-tools/internal/servers"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"text/tabwriter"
)

var Verbose bool
var IncludeUnkown bool

func newStrandedCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stranded [OPTIONS]",
		Short: "Check for inactive (stranded) apps",
		Long: `Check for inactive (stranded) apps. An app is considered stranded
  if it exists on the server but does not have DNS records pointing to it.
  This makes it easy to find apps that are no longer in use or have migrated
  away and can be deleted.`,
		Args: cobra.ExactArgs(2),
		//PreRunE: func(cmd *cobra.Command, args []string) error {
		//	// Validate here?
		//},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := log.New(io.Discard, "", 0)
			if Verbose {
				logger.SetOutput(os.Stdout)
			}

			c := serverpilot.NewClient(args[0], args[1])

			// Get all servers, and extract their ip addresses
			s, err := servers.GetServers(c)
			if err != nil {
				return fmt.Errorf("error while getting servers: %w", err)
			}
			var serverIps []string
			for _, server := range s {
				serverIps = append(serverIps, server.Ipaddress)
			}

			// Get all ServerPilot apps
			apps, err := filter.FilterApps(c, "", "", 0, 0)
			if err != nil {
				return fmt.Errorf("error while getting apps: %w", err)
			}

			var domainToStatus map[string]int
			domainToStatus = make(map[string]int)

			dnsChecker := dns.NewDnsChecker(dns.NewResolver(nil, nil, logger), serverIps)

			// Loop through each domain, and check if it resolves to the server
			for _, app := range apps {
				for _, domain := range app.Domains {
					status := dnsChecker.CheckStatus(domain)

					domainToStatus[domain] = status
				}
			}

			// Only print out the stranded apps by default, but allow the user to include unknown domains with a flag
			var domainsToPrint map[string]int
			domainsToPrint = make(map[string]int)
			for domain, status := range domainToStatus {
				if status == dns.STRANDED {
					domainsToPrint[domain] = status
				}

				if IncludeUnkown && status == dns.UNKNOWN {
					domainsToPrint[domain] = status
				}
			}

			// Print out the stranded apps, with their status (STRANDED/PARTIAL/UNKNOWN)
			printDomains(domainsToPrint)

			return nil
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&Verbose, "verbose", "v", false, "Verbose output")
	flags.BoolVarP(&IncludeUnkown, "include-unknown", "u", false, "Include domains with unknown status")

	return cmd
}

func printDomains(domains map[string]int) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "DOMAIN\tSTATUS\t")
	for domain, status := range domains {
		stringStatus := ""
		switch status {
		case dns.OK:
			stringStatus = "ok"
		case dns.STRANDED:
			stringStatus = "stranded"
		case dns.UNKNOWN:
			stringStatus = "unknown"
		}
		fmt.Fprintln(w, domain+"\t"+stringStatus+"\t")
	}
	w.Flush()
}
