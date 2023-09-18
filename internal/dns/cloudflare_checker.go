package dns

import (
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
)

// CloudflareCredentialsChecker is responsible for checking a list of domains and determining if they are behind Cloudflare. It will prompt for API credentials for each unique cloudflare account detected.
type CloudflareCredentialsChecker struct {
	l        *log.Logger
	p        CredentialsPrompter
	lookupNs NsLookupFunc
	cachedNs map[string][]string
}

// CredentialsPrompter is an interface for interacting with the user to prompt for input
// It optionally takes a slice of valid responses and won't return until the user enters a valid response
type CredentialsPrompter interface {
	Prompt(msg, defaultResponse string, validResponse []string) string
}

type Prompter struct {
}

func (p *Prompter) Prompt(msg, defaultResponse string, validResponse []string) string {
	fmt.Print(msg)
	var response string
	fmt.Scanln(&response)

	// If the response is empty, then use the default response
	if response == "" {
		response = defaultResponse
	}

	// If the response is not in the valid responses, then prompt again
	for validResponse != nil && !contains(validResponse, response) {
		return p.Prompt(msg, defaultResponse, validResponse)
	}

	return response
}

// NameserverDomains is a list of domains that are managed by the same Cloudflare account. The API credentials are used to authenticate with the Cloudflare API.
type NameserverDomains struct {
	Nameservers []string
	Domains     []string
	Credentials *Credentials
}

func NewCloudflareCredentialsChecker(l *log.Logger, p CredentialsPrompter, nsLookup NsLookupFunc) *CloudflareCredentialsChecker {
	// Default to net.LookupNS
	if nsLookup == nil {
		nsLookup = net.LookupNS
	}

	return &CloudflareCredentialsChecker{l: l, p: p, lookupNs: nsLookup}
}

// IsBehindCloudFlare checks if the domain is behind CloudFlare by looking up the nameservers for the base domain.
func (c *CloudflareCredentialsChecker) IsBehindCloudFlare(domain string) bool {
	ns, _ := c.GetNameserversForBaseDomain(domain)

	for _, n := range ns {
		// Check if the nameserver format matches *.ns.cloudflare.com
		if len(n) >= 17 && n[len(n)-17:] == "ns.cloudflare.com" {
			return true
		}
	}

	return false
}

// GetNameserversForBaseDomain looks up the nameservers for the base domain. It caches the nameservers for each domain so additional lookups are not needed.
func (c *CloudflareCredentialsChecker) GetNameserversForBaseDomain(domain string) ([]string, error) {
	baseDomain := getBaseDomain(domain)

	// Check if we've already looked up the nameservers for this domain
	if c.cachedNs == nil || c.cachedNs[baseDomain] == nil {
		c.l.Println("Looking up nameservers for", baseDomain, "...")
		ns, _ := c.lookupNs(baseDomain)
		var nsStrings []string
		for _, n := range ns {
			// Remove trailing dot
			host := strings.TrimSuffix(n.Host, ".")
			nsStrings = append(nsStrings, host)
		}
		// Sort the nameservers so that we can compare them later
		sort.Strings(nsStrings)
		c.l.Println("Nameservers for", baseDomain, "are", nsStrings)

		// Cache the nameservers for this domain
		// Initialize the map if it's nil
		if c.cachedNs == nil {
			c.cachedNs = make(map[string][]string)
		}
		c.cachedNs[baseDomain] = nsStrings
	}

	return c.cachedNs[baseDomain], nil
}

func (c *CloudflareCredentialsChecker) checkDomains(domains []UnresolvedDomain) ([]NameserverDomains, error) {
	nameserverDomains := make([]NameserverDomains, 0)

	for _, domain := range domains {
		if domain.IsBehindCloudflare {
			nameserverDomains = appendOrInitializeNameserverDomains(nameserverDomains, domain.BaseDomainNameservers, domain.Name)
		}
	}

	return nameserverDomains, nil
}

func (c *CloudflareCredentialsChecker) PromptForCredentials(domains []UnresolvedDomain) []UnresolvedDomain {
	nameserverDomains, err := c.checkDomains(domains)
	if err != nil {
		return nil
	}

	validYesNoResponses := []string{"y", "Y", "n", "N"}

	// The first thing we want it to say is the number of accounts detected, and ask if they want to enter credentials
	response := c.p.Prompt(fmt.Sprintf("Detected %v CloudFlare accounts. Do you want to use the CloudFlare API to check DNS records? [y/N]", len(nameserverDomains)), "N", validYesNoResponses)

	// If they say no, then we should just return the original domains
	if response == "n" || response == "N" {
		return domains
	}

	// Then the prompter should be called for each unique nameserver
	result := make([]NameserverDomains, 0)

	for _, nsd := range nameserverDomains {
		creds := c.promptForCredentials(nsd)
		nsd.Credentials = creds
		result = append(result, nsd)
	}

	// Loop through all the domains and set the matching credentials
	for i, domain := range domains {
		for _, nsd := range result {
			if contains(nsd.Domains, domain.Name) {
				domains[i].CloudflareCredentials = nsd.Credentials
				break
			}
		}
	}

	return domains
}

func (c *CloudflareCredentialsChecker) promptForCredentials(nsd NameserverDomains) *Credentials {
	validYesNoResponses := []string{"y", "Y", "n", "N"}

	ns := strings.Join(nsd.Nameservers, ", ")
	response := c.p.Prompt(fmt.Sprintf("Would you like to enter credentials for %s (domains: %s)? [y/N]", ns, getDomainsText(nsd.Domains, 3)), "N", validYesNoResponses)

	// If they say no, then we should just return nil, nil
	if response == "n" || response == "N" {
		return nil
	}

	email := c.p.Prompt(fmt.Sprintf("Email:"), "", nil)
	token := c.p.Prompt(fmt.Sprintf("API Token:"), "", nil)

	//shouldStore := c.p.Prompt(fmt.Sprintf("Store these credentials? [y/N]"), "N", validYesNoResponses)
	//
	//if shouldStore == "y" || shouldStore == "Y" {
	//}

	return &Credentials{email, token}
}
