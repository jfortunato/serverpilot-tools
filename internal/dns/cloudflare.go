package dns

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/http"
	"log"
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
	l              *log.Logger
	c              http.CachingRateLimitedClient
	creds          *Credentials
	nameservers    []string
	cachedRecoreds []DnsRecord
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
func NewCloudflareResolver(l *log.Logger, c http.CachingRateLimitedClient, creds *Credentials, nameservers []string) *CloudflareResolver {
	return &CloudflareResolver{l: l, c: c, creds: creds, nameservers: nameservers}
}

// Resolve resolves the domain using the Cloudflare API. It implements the IpResolver interface.
// We cache the DNS records, even though the http requests are also cached by the http.CachingRateLimitedClient, because
// it's faster to read from memory than to read from disk.
func (r *CloudflareResolver) Resolve(domain string) ([]string, error) {
	if r.creds == nil {
		return nil, fmt.Errorf("%w: no credentials provided", ErrCouldNotMakeRequest)
	}

	// Get all DNS records for the account, and cache them
	if r.cachedRecoreds == nil {
		r.cachedRecoreds = r.getDnsRecords()
	}

	records := r.cachedRecoreds

	return findMatchingRecord(domain, records), nil
}

func findMatchingRecord(domain string, records []DnsRecord) []string {
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

				return findMatchingRecord(target, records)
			}

			if record.Type == "A" {
				matched = append(matched, record.Content)
			}
		}
	}

	return matched
}

func (r *CloudflareResolver) getDnsRecords() []DnsRecord {
	records, err := r.getDnsRecordsForAllZones()
	if err != nil {
		r.l.Printf("%s", fmt.Errorf("%w: %s", ErrCouldNotMakeRequest, err))
		return nil
	}

	return records
}

func (r *CloudflareResolver) getDnsRecordsForAllZones() ([]DnsRecord, error) {
	zones, err := r.getZones()
	if err != nil {
		return nil, err
	}

	var records []DnsRecord

	for _, zone := range zones {
		recordsForZone, err := r.getDnsRecordsForZone(zone)
		if err != nil {
			return nil, err
		}

		for _, record := range recordsForZone {
			records = append(records, record)
		}
	}

	return records, nil
}

func (r *CloudflareResolver) makeCloudflareRequest(url string) http.Request {
	headers := map[string]string{
		"X-Auth-Email": r.creds.Email,
		"X-Auth-Key":   r.creds.ApiToken,
		"Content-Type": "application/json",
	}

	return http.Request{url, headers}
}

func (r *CloudflareResolver) getZones() ([]Zone, error) {
	endpoint := "https://api.cloudflare.com/client/v4/zones"

	page := 1
	haveMadeRequest := false
	var lastResponse CloudflareResponse[[]Zone]

	var items []Zone

	for !haveMadeRequest || lastResponse.ResultInfo.Page < lastResponse.ResultInfo.TotalPages {
		request := r.makeCloudflareRequest(fmt.Sprintf("%s?page=%d&per_page=%d", endpoint, page, PerPage))
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

func (r *CloudflareResolver) getDnsRecordsForZone(z Zone) ([]DnsRecord, error) {
	endpoint := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", z.Id)

	page := 1
	haveMadeRequest := false
	var lastResponse CloudflareResponse[[]DnsRecord]

	var items []DnsRecord

	for !haveMadeRequest || lastResponse.ResultInfo.Page < lastResponse.ResultInfo.TotalPages {
		request := r.makeCloudflareRequest(fmt.Sprintf("%s?page=%d&per_page=%d", endpoint, page, PerPage))
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
