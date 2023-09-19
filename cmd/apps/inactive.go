package apps

import (
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/dns"
	"github.com/jfortunato/serverpilot-tools/internal/filter"
	"github.com/jfortunato/serverpilot-tools/internal/progressbar"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"github.com/jfortunato/serverpilot-tools/internal/servers"
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
	cfChecker := dns.NewCloudflareCredentialsChecker(logger, &dns.Prompter{}, nil)
	dnsChecker := createDomainChecker(logger, cfChecker)

	apps, err := getAppServers(logger, user, key)
	if err != nil {
		return err
	}

	// Transform the list of AppServers into a list of all their domains
	domains := getAllDomains(apps)

	bar := progressbar.NewProgressBar(len(domains), "Evaluating domains")

	// Evaluate all domains, and determine if they are behind Cloudflare
	unresolvedDomains := dnsChecker.EvaluateDomains(bar, domains)

	bar.Finish()

	// Prompt for Cloudflare credentials for each unique account discovered
	unresolvedDomains = cfChecker.PromptForCredentials(unresolvedDomains)

	// Resolve the UnresolvedDomains, and determine if they are pointing to the correct server
	// Only print out the inactive apps by default, but allow the user to include unknown domains with a flag
	bar = progressbar.NewProgressBar(len(unresolvedDomains), "Checking domains")
	filtered := dnsChecker.GetInactiveAppDomains(bar, unresolvedDomains, apps, options.includeUnknown)
	bar.Finish()
	bar.Clear()

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

func getAllDomains(appServers []serverpilot.AppServer) []string {
	var domains []string
	for _, appServer := range appServers {
		domains = append(domains, appServer.Domains...)
	}
	return domains
}

func createLogger(isVerbose bool) *log.Logger {
	logger := log.New(io.Discard, "", 0)
	if isVerbose {
		logger.SetOutput(os.Stdout)
	}
	return logger
}

func createDomainChecker(logger *log.Logger, checker *dns.CloudflareCredentialsChecker) *dns.DnsChecker {
	return dns.NewDnsChecker(dns.NewResolver(nil, checker, nil, logger), checker)
}

func getServerForApp(app serverpilot.App, servers []serverpilot.Server) serverpilot.Server {
	for _, server := range servers {
		if server.Id == app.Serverid {
			return server
		}
	}

	return serverpilot.Server{}
}

func printDomains(domains []dns.AppDomainStatus) error {
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
