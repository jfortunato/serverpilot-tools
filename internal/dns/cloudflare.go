package dns

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/http"
	"log"
	"net"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

var (
	ErrCouldNotMakeRequest = errors.New("could not make request to Cloudflare API")
)

// PerPage How many zones/records to fetch per page.
const PerPage = 50

// CloudflareResolver is a DNS resolver that uses the Cloudflare API to resolve DNS records.
type CloudflareResolver struct {
	l                 *log.Logger
	parent            IpResolver
	c                 http.CachingRateLimitedClient
	nameserverDomains []NameserverDomains
}

// Credentials are the credentials used to authenticate with the Cloudflare API.
type Credentials struct {
	Email    string
	ApiToken string
}

// Zone is a Cloudflare zone.
type Zone struct {
	Id string `json:"id"`
}

// DnsRecord is a Cloudflare DNS record.
type DnsRecord struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ResultInfo is the result info returned by the Cloudflare API. It is used to determine if there are more pages of results.
type ResultInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
	Count      int `json:"count"`
	TotalCount int `json:"total_count"`
}

// CodedMessage is a message returned by the Cloudflare API.
type CodedMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CloudflareResponse is a response returned by the Cloudflare API. The generic type T is the type of the result ([]Zone or []DnsRecord).
type CloudflareResponse[T any] struct {
	ResultInfo ResultInfo     `json:"result_info"`
	Result     T              `json:"result"`
	Success    bool           `json:"success"`
	Errors     []CodedMessage `json:"errors"`
	Messages   []CodedMessage `json:"messages"`
}

// NewCloudflareResolver creates a new CloudflareResolver. Caching and rate limiting of the API requests is handled by the http.CachingRateLimitedClient.
// The nameservers are used to determine if the domain is managed by the Cloudflare account that we have credentials for.
func NewCloudflareResolver(l *log.Logger, parent IpResolver, c http.CachingRateLimitedClient, nsd []NameserverDomains) *CloudflareResolver {
	return &CloudflareResolver{
		l:                 l,
		parent:            parent,
		c:                 c,
		nameserverDomains: nsd,
	}
}

// Resolve resolves the domain using the Cloudflare API. It implements the IpResolver interface.
func (r *CloudflareResolver) Resolve(domain string) ([]string, error) {
	creds := r.getCredentialsForDomain(domain)

	if creds == nil {
		return nil, fmt.Errorf("%w: no credentials provided", ErrCouldNotMakeRequest)
	}

	zone, err := r.getZoneForDomain(domain, creds)
	if err != nil {
		return nil, err
	}

	records, err := r.getDnsRecordsForZone(zone, creds)
	if err != nil {
		return nil, err
	}

	return r.findMatchingRecord(domain, records)
}

func (r *CloudflareResolver) getCredentialsForDomain(domain string) *Credentials {
	for _, nsd := range r.nameserverDomains {
		if contains(nsd.Domains, domain) {
			return nsd.Credentials
		}
	}

	return nil
}

func (r *CloudflareResolver) findMatchingRecord(domain string, records []DnsRecord) ([]string, error) {
	var matched []string

	// Find the DNS record for the domain
	for _, record := range records {
		// For the regex, we need to escape the dots in the domain name and replace the asterisks with a regex wildcard
		// Escape the dots
		recordName := strings.Replace(record.Name, ".", `\.`, -1)
		// Replace the asterisks with a regex wildcard
		recordName = strings.Replace(recordName, "*", ".+", -1)

		regex, _ := regexp.Compile(fmt.Sprintf("^%s$", recordName))

		if regex.MatchString(domain) {
			if record.Type == "CNAME" {
				target := record.Content

				// If the target is for the same base domain, then re-check the records for a matching A record
				if getBaseDomain(target) == getBaseDomain(domain) {
					return r.findMatchingRecord(target, records)
				}

				return r.parent.Resolve(target)
			}

			if record.Type == "A" {
				matched = append(matched, record.Content)
			}
		}
	}

	return matched, nil
}

func (r *CloudflareResolver) getZoneForDomain(domain string, creds *Credentials) (Zone, error) {
	baseDomain := getBaseDomain(domain)
	endpoint := "https://api.cloudflare.com/client/v4/zones?name=" + baseDomain
	request := r.makeCloudflareRequest(endpoint, creds)
	contents, err := r.c.GetFromCacheOrFetchWithRateLimit(request)
	if err != nil {
		return Zone{}, err
	}
	cloudflareResponse := CloudflareResponse[[]Zone]{}
	// Unmarshal the response into a CloudflareResponse
	err = json.Unmarshal([]byte(contents), &cloudflareResponse)
	if err != nil {
		return Zone{}, fmt.Errorf("error while unmarshalling response body: %s", err)
	}

	if len(cloudflareResponse.Result) > 0 {
		return cloudflareResponse.Result[0], nil
	}

	return Zone{}, fmt.Errorf("no zone found for domain %s", domain)
}

func (r *CloudflareResolver) makeCloudflareRequest(url string, creds *Credentials) http.Request {
	headers := map[string]string{
		"X-Auth-Email": creds.Email,
		"X-Auth-Key":   creds.ApiToken,
		"Content-Type": "application/json",
	}

	return http.Request{url, headers}
}

func (r *CloudflareResolver) getDnsRecordsForZone(z Zone, creds *Credentials) ([]DnsRecord, error) {
	endpoint := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", z.Id)

	page := 1
	haveMadeRequest := false
	var lastResponse CloudflareResponse[[]DnsRecord]

	var items []DnsRecord

	for !haveMadeRequest || lastResponse.ResultInfo.Page < lastResponse.ResultInfo.TotalPages {
		request := r.makeCloudflareRequest(fmt.Sprintf("%s?page=%d&per_page=%d", endpoint, page, PerPage), creds)
		contents, err := r.c.GetFromCacheOrFetchWithRateLimit(request)
		if err != nil {
			return nil, err
		}
		haveMadeRequest = true
		// Unmarshal the response into a CloudflareResponse
		err = json.Unmarshal([]byte(contents), &lastResponse)
		if err != nil {
			return nil, fmt.Errorf("error while unmarshalling response body: %s", err)
		}

		for _, item := range lastResponse.Result {
			items = append(items, item)
		}

		page++
	}

	return items, nil
}

// CloudflareCredentialsChecker is responsible for checking a list of domains and determing if they are behind Cloudflare. It will prompt for API credentials for each unique cloudflare account detected.
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

func (c *CloudflareCredentialsChecker) checkDomains(domains []string) ([]NameserverDomains, error) {
	nameserverDomains := make([]NameserverDomains, 0)

	for _, domain := range domains {
		if c.IsBehindCloudFlare(domain) {
			nameservers, err := c.GetNameserversForBaseDomain(domain)
			if err != nil {
				return nil, err
			}
			nameserverDomains = appendOrInitializeNameserverDomains(nameserverDomains, nameservers, domain)
		}
	}

	return nameserverDomains, nil
}

func (c *CloudflareCredentialsChecker) PromptForCredentials(domains []string) []NameserverDomains {
	nameserverDomains, err := c.checkDomains(domains)
	if err != nil {
		return nil
	}

	validYesNoResponses := []string{"y", "Y", "n", "N"}

	// The first thing we want it to say is the number of accounts detected, and ask if they want to enter credentials
	response := c.p.Prompt(fmt.Sprintf("Detected %v CloudFlare accounts. Do you want to use the CloudFlare API to check DNS records? [y/N]", len(nameserverDomains)), "N", validYesNoResponses)

	// If they say no, then we should just return nil, nil
	if response == "n" || response == "N" {
		return nameserverDomains
	}

	// Then the prompter should be called for each unique nameserver
	result := make([]NameserverDomains, 0)

	for _, nsd := range nameserverDomains {
		creds := c.promptForCredentials(nsd)
		nsd.Credentials = creds
		result = append(result, nsd)
	}

	return result
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

	return &Credentials{email, token}
}

func getDomainsText(domains []string, totalToShow int) string {
	var firstFewDomains []string

	if len(domains) > totalToShow {
		firstFewDomains = domains[:totalToShow]
	} else {
		firstFewDomains = domains
	}

	totalRest := len(domains) - totalToShow

	domainsText := strings.Join(firstFewDomains, ", ")

	if totalRest > 0 {
		domainsText = fmt.Sprintf("%s, and %d more", domainsText, totalRest)
	}

	return domainsText
}

func appendOrInitializeNameserverDomains(existing []NameserverDomains, nameservers []string, domain string) []NameserverDomains {
	for index, item := range existing {
		if reflect.DeepEqual(item.Nameservers, nameservers) {
			existing[index].Domains = append(item.Domains, domain)
			return existing
		}
	}

	return append(existing, NameserverDomains{Nameservers: nameservers, Domains: []string{domain}})
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}

	return false
}
