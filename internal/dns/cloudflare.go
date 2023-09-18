package dns

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/http"
	"log"
	"reflect"
	"regexp"
	"strings"
)

var (
	ErrCouldNotMakeRequest = errors.New("could not make request to Cloudflare API")
)

// PerPage How many zones/records to fetch per page.
const PerPage = 50

// CloudflareResolver is a DNS resolver that uses the Cloudflare API to resolve DNS records.
type CloudflareResolver struct {
	l      *log.Logger
	parent IpResolver
	c      http.CachingRateLimitedClient
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
func NewCloudflareResolver(l *log.Logger, parent IpResolver, c http.CachingRateLimitedClient) *CloudflareResolver {
	return &CloudflareResolver{
		l:      l,
		parent: parent,
		c:      c,
	}
}

// Resolve resolves the domain using the Cloudflare API. It implements the IpResolver interface.
func (r *CloudflareResolver) Resolve(domain UnresolvedDomain) ([]string, error) {
	creds := domain.CloudflareCredentials

	if creds == nil {
		return nil, fmt.Errorf("%w: no credentials provided", ErrCouldNotMakeRequest)
	}

	zone, err := r.getZoneForDomain(domain.Name, creds)
	if err != nil {
		return nil, err
	}

	records, err := r.getDnsRecordsForZone(zone, creds)
	if err != nil {
		return nil, err
	}

	return r.findMatchingRecord(domain.Name, records)
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

				return r.parent.Resolve(UnresolvedDomain{
					Name:                  target,
					IsBehindCloudflare:    false,
					BaseDomainNameservers: nil,
					CloudflareCredentials: nil,
				})
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
