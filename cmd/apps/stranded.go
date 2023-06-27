package apps

import (
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/dns"
	"github.com/jfortunato/serverpilot-tools/internal/filter"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"strings"
	"text/tabwriter"
)

var Servers string
var Verbose bool

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

			// Get all ServerPilot apps
			apps, _ := filter.FilterApps(c, "", "", 0, 0)

			var domainToStatus map[string]int
			domainToStatus = make(map[string]int)

			dnsChecker := dns.NewDnsChecker(dns.NewResolver(nil, nil, logger), strings.Split(Servers, " "))

			// Loop through each domain, and check if it resolves to the server
			for _, app := range apps {
				for _, domain := range app.Domains {
					status := dnsChecker.CheckStatus(domain)

					domainToStatus[domain] = status
				}
			}

			// Print out the stranded apps, with their status (STRANDED/PARTIAL/UNKNOWN)
			printDomains(domainToStatus)

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&Servers, "servers", "", "Ip address of the server to check against")
	flags.BoolVarP(&Verbose, "verbose", "v", false, "Verbose output")

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
