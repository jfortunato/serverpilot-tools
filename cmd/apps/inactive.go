package apps

import (
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/dns"
	"github.com/jfortunato/serverpilot-tools/internal/filter"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"github.com/jfortunato/serverpilot-tools/internal/servers"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"strings"
	"text/tabwriter"
)

var Verbose bool
var IncludeUnknown bool
var CloudflareCredentials string

func newInactiveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inactive [OPTIONS]",
		Short: "Check for inactive (stranded) apps",
		Long: `Check for inactive (stranded) apps. An app is considered inactive
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

			c := serverpilot.NewClient(logger, args[0], args[1])

			// Get all servers, and extract their ip addresses
			s, err := servers.GetServers(c)
			if err != nil {
				return fmt.Errorf("error while getting servers: %w", err)
			}

			// Get all ServerPilot apps
			apps, err := filter.FilterApps(c, "", "", 0, 0)
			if err != nil {
				return fmt.Errorf("error while getting apps: %w", err)
			}

			var creds *dns.Credentials
			if CloudflareCredentials != "" {
				split := strings.Split(CloudflareCredentials, ":")
				creds = &dns.Credentials{split[0], split[1]}
			}

			dnsChecker := dns.NewDnsChecker(dns.NewResolver(creds, nil, nil, logger))

			bar := progressbar.Default(int64(len(apps)))

			var appDomains []AppDomainStatus

			// Loop through each domain, and check if it resolves to the server
			for _, app := range apps {
				for _, domain := range app.Domains {
					serverForApp := getServerForApp(app, s)

					status := dnsChecker.CheckStatus(domain, serverForApp.Ipaddress)

					appDomains = append(appDomains, AppDomainStatus{app.Id, domain, serverForApp.Name, status})
				}
				bar.Add(1)
			}

			bar.Clear()

			// Only print out the inactive apps by default, but allow the user to include unknown domains with a flag
			var filtered []AppDomainStatus
			for _, appDomain := range appDomains {
				if appDomain.Status == dns.INACTIVE {
					filtered = append(filtered, appDomain)
				}

				if IncludeUnknown && appDomain.Status == dns.UNKNOWN {
					filtered = append(filtered, appDomain)
				}
			}

			// Print out the inactive apps, with their status (INACTIVE/PARTIAL/UNKNOWN)
			printDomains(filtered)

			return nil
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&Verbose, "verbose", "v", false, "Verbose output")
	flags.BoolVarP(&IncludeUnknown, "include-unknown", "u", false, "Include domains with unknown status")
	flags.StringVarP(&CloudflareCredentials, "cloudflare-credentials", "", "", "Cloudflare credentials (email:api-key)")

	return cmd
}

type AppDomainStatus struct {
	AppId      string
	Domain     string
	ServerName string
	Status     int
}

func getServerForApp(app serverpilot.App, servers []serverpilot.Server) serverpilot.Server {
	for _, server := range servers {
		if server.Id == app.Serverid {
			return server
		}
	}

	return serverpilot.Server{}
}

func printDomains(domains []AppDomainStatus) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "APP ID\tDOMAIN\tSERVER\tSTATUS\t")
	for _, domain := range domains {
		stringStatus := ""
		switch domain.Status {
		case dns.OK:
			stringStatus = "ok"
		case dns.INACTIVE:
			stringStatus = "inactive"
		case dns.UNKNOWN:
			stringStatus = "unknown"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", domain.AppId, domain.Domain, domain.ServerName, stringStatus)
	}
	w.Flush()
}
