package dns

import (
	"github.com/jfortunato/serverpilot-tools/internal/progressbar"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"golang.org/x/net/publicsuffix"
	"strings"
)

const (
	OK int = iota
	INACTIVE
	UNKNOWN
)

type DnsChecker struct {
	r         IpResolver
	cfChecker *CloudflareCredentialsChecker
}

func NewDnsChecker(r IpResolver, cfChecker *CloudflareCredentialsChecker) *DnsChecker {
	return &DnsChecker{
		r,
		cfChecker,
	}
}

type AppDomainStatus struct {
	AppId      string
	Domain     string
	ServerName string
	Status     int
}

// UnresolvedDomain is the result of evaluating a domain's metadata, before it is resolved.
type UnresolvedDomain struct {
	Name               string
	CloudflareMetadata *CloudflareDomainMetadata
}

func (c *DnsChecker) EvaluateDomains(domains []string) []UnresolvedDomain {
	var results []UnresolvedDomain

	for _, domain := range domains {
		var cloudflareMetadata *CloudflareDomainMetadata
		if c.cfChecker.IsBehindCloudFlare(domain) {
			// Only get the nameservers if the domain is behind Cloudflare
			ns, _ := c.cfChecker.GetNameserversForBaseDomain(domain)
			cloudflareMetadata = &CloudflareDomainMetadata{
				BaseDomainNameservers: ns,
				CloudflareCredentials: nil, // Will be prompted for later
			}
		}

		result := UnresolvedDomain{
			Name:               domain,
			CloudflareMetadata: cloudflareMetadata,
		}

		results = append(results, result)
	}

	return results
}

// GetInactiveAppDomains will return a list of domains that are not resolving to the server they are
// assigned to.
func (c *DnsChecker) GetInactiveAppDomains(ticker progressbar.Ticker, domains []UnresolvedDomain, appservers []serverpilot.AppServer, includeUnknown bool) []AppDomainStatus {
	var results []AppDomainStatus

	// Loop through each domain, and check if it resolves to the server
	for _, domain := range domains {
		// Find the appserver that matches the domain
		appserver := findMatchingAppServer(domain, appservers)

		status := c.CheckStatus(domain, appserver.Server.Ipaddress)

		if status == INACTIVE || (includeUnknown && status == UNKNOWN) {
			results = append(results, AppDomainStatus{appserver.Id, domain.Name, appserver.Server.Name, status})
		}

		// Tick the progress bar
		ticker.Tick()
	}

	return results
}

func findMatchingAppServer(domain UnresolvedDomain, appservers []serverpilot.AppServer) serverpilot.AppServer {
	for _, appserver := range appservers {
		for _, appdomain := range appserver.Domains {
			if appdomain == domain.Name {
				return appserver
			}
		}
	}

	return serverpilot.AppServer{}
}

func (c *DnsChecker) CheckStatus(domain UnresolvedDomain, serverIp string) int {
	resolvedIps, err := c.r.Resolve(domain)
	if err != nil {
		return UNKNOWN
	}

	for _, ip := range resolvedIps {
		if ip == serverIp {
			return OK
		}
	}

	return INACTIVE
}

func getBaseDomain(domain string) string {
	// Get the effective top level domain (eTLD)
	eTLD, _ := publicsuffix.PublicSuffix(domain)

	// Get the domain without the eTLD
	domainWithoutTld := strings.TrimSuffix(domain, "."+eTLD)

	// Split the domain by the dot character
	parts := strings.Split(domainWithoutTld, ".")

	// The last part is the base domain
	return parts[len(parts)-1] + "." + eTLD
}

// IpResolver is an interface for resolving a domain to its IP address(s). It will return
// the ip addresses when it can, or an error if it cannot.
type IpResolver interface {
	Resolve(domain UnresolvedDomain) ([]string, error)
}
