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
	"text/tabwriter"
)

type inactiveOptions struct {
	verbose        bool
	includeUnknown bool
}

func newInactiveCommand() *cobra.Command {
	options := inactiveOptions{}

	cmd := &cobra.Command{
		Use:   "inactive [OPTIONS]",
		Short: "Check for inactive (stranded) apps",
		Long: `Check for inactive (stranded) apps. An app is considered inactive
  if it exists on the server but does not have DNS records pointing to it.
  This makes it easy to find apps that are no longer in use or have migrated
  away and can be deleted.`,
		Args: cobra.ExactArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInactive(args[0], args[1], options)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.verbose, "verbose", "v", false, "Verbose output")
	flags.BoolVarP(&options.includeUnknown, "include-unknown", "u", false, "Include domains with unknown status")

	return cmd
}

func runInactive(user, key string, options inactiveOptions) error {
	logger := createLogger(options.verbose)
	cloudflareChecker := createCloudflareChecker(logger)

	var appDomains []AppDomainStatus

	apps, err := getAppServers(logger, user, key)
	if err != nil {
		return err
	}

	// Get all domains
	domains := make([]string, 0)
	for _, app := range apps {
		for _, domain := range app.Domains {
			domains = append(domains, domain)
		}
	}

	nsd := cloudflareChecker.PromptForCredentials(domains)
	dnsChecker := createDomainChecker(logger, cloudflareChecker, nsd)

	bar := progressbar.Default(int64(len(apps)))

	// Loop through each domain, and check if it resolves to the server
	for _, app := range apps {
		for _, domain := range app.Domains {
			status := dnsChecker.CheckStatus(domain, app.Server.Ipaddress)

			appDomains = append(appDomains, AppDomainStatus{app.Id, domain, app.Server.Name, status})
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

		if options.includeUnknown && appDomain.Status == dns.UNKNOWN {
			filtered = append(filtered, appDomain)
		}
	}

	// Print out the inactive apps, with their status (INACTIVE/PARTIAL/UNKNOWN)
	return printDomains(filtered)
}

func getAppServers(logger *log.Logger, user, key string) ([]serverpilot.AppServer, error) {
	c := serverpilot.NewClient(logger, user, key)

	// Get all servers, and extract their ip addresses
	srvers, err := servers.GetServers(c)
	if err != nil {
		return nil, fmt.Errorf("error while getting servers: %w", err)
	}

	// Get all ServerPilot apps
	apps, err := filter.FilterApps(c, "", "", 0, 0)
	if err != nil {
		return nil, fmt.Errorf("error while getting apps: %w", err)
	}

	var appServers []serverpilot.AppServer

	// Add the matching server to each app
	for _, app := range apps {
		server := getServerForApp(app, srvers)
		appServers = append(appServers, serverpilot.AppServer{app, server})
	}

	return appServers, nil
}

func createLogger(isVerbose bool) *log.Logger {
	logger := log.New(io.Discard, "", 0)
	if isVerbose {
		logger.SetOutput(os.Stdout)
	}
	return logger
}

func createCloudflareChecker(logger *log.Logger) *dns.CloudflareCredentialsChecker {
	return dns.NewCloudflareCredentialsChecker(logger, &dns.Prompter{}, nil)
}

func createDomainChecker(logger *log.Logger, checker *dns.CloudflareCredentialsChecker, nsd []dns.NameserverDomains) *dns.DnsChecker {
	return dns.NewDnsChecker(dns.NewResolver(nil, checker, nsd, nil, logger))
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

func printDomains(domains []AppDomainStatus) error {
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
	return w.Flush()
}
