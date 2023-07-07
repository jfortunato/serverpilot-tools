package dns

import (
	"encoding/json"
	"errors"
	"github.com/jfortunato/serverpilot-tools/internal/http"
	"gotest.tools/v3/assert"
	"io"
	"log"
	"net"
	"testing"
)

func TestCloudflare(t *testing.T) {
	t.Run("it should not be able to resolve a domain behind CloudFlare nameservers that we don't have api credentials for", func(t *testing.T) {
		resolver := newCloudflareResolverWithStubs()

		got, _ := resolver.Resolve("domain-behind-cloudflare.com")

		assert.Assert(t, got == nil)
	})

	t.Run("it should be able to resolve a domain behind CloudFlare nameservers that we do have api credentials for", func(t *testing.T) {
		var tests = []struct {
			name    string
			domain  string
			records []DnsRecord
			want    []string
		}{
			{
				"matches domain and type",
				"example.com",
				[]DnsRecord{
					{"A", "example.com", "127.0.0.1"},
				},
				[]string{"127.0.0.1"},
			},
			{
				"matches domain but not type",
				"example.com",
				[]DnsRecord{
					{"TXT", "example.com", "127.0.0.1"},
				},
				nil,
			},
			{
				"matches type but not domain",
				"sub.example.com",
				[]DnsRecord{
					{"A", "example.com", "127.0.0.1"},
				},
				nil,
			},
			{
				"matches domain and type - multiple records",
				"example.com",
				[]DnsRecord{
					{"A", "example.com", "127.0.0.1"},
					{"A", "example.com", "127.0.0.2"},
				},
				[]string{"127.0.0.1", "127.0.0.2"},
			},
			{
				"matched domain is a CNAME - matches A record",
				"www.example.com",
				[]DnsRecord{
					{"CNAME", "www.example.com", "example.com"},
					{"A", "example.com", "127.0.0.1"},
				},
				[]string{"127.0.0.1"},
			},
			{
				"matched domain is a wildcard CNAME - matches A record",
				"www.example.com",
				[]DnsRecord{
					{"CNAME", "*.example.com", "example.com"},
					{"A", "example.com", "127.0.0.1"},
				},
				[]string{"127.0.0.1"},
			},
			{
				"matched domain is a wildcard A record",
				"www.example.com",
				[]DnsRecord{
					{"A", "*.example.com", "127.0.0.1"},
				},
				[]string{"127.0.0.1"},
			},
		}

		// These tests will always use the same zone response
		stubbedZoneResponses := makeStubbedZoneResponse("https://api.cloudflare.com/client/v4/zones?name=example.com", []Zone{{"1"}})

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				stubbedDnsResponses := makeStubbedDnsResponse("https://api.cloudflare.com/client/v4/zones/1/dns_records?page=1&per_page=50", tt.records)

				resolver := newCloudflareResolverWithStubs()
				// Combine the two responses into a single map
				resolver.c = &ClientStub{responses: combineResponses(stubbedZoneResponses, stubbedDnsResponses)}

				got, _ := resolver.Resolve(tt.domain)

				assert.DeepEqual(t, got, tt.want)
			})
		}
	})

	t.Run("it should return an error when no zone is found for the base domain", func(t *testing.T) {
		stubbedZoneResponse := makeStubbedZoneResponse("https://api.cloudflare.com/client/v4/zones?name=example.com", []Zone{})

		resolver := newCloudflareResolverWithStubs()
		resolver.c = &ClientStub{responses: stubbedZoneResponse}

		got, err := resolver.Resolve("example.com")

		assert.Assert(t, got == nil)
		assert.ErrorContains(t, err, "no zone found for domain")
	})

	t.Run("it should resolve the dns for a cname not within the same zone", func(t *testing.T) {
		stubbedZoneResponses := makeStubbedZoneResponse("https://api.cloudflare.com/client/v4/zones?name=example.com", []Zone{{"1"}})

		stubbedDnsResponses := makeStubbedDnsResponse("https://api.cloudflare.com/client/v4/zones/1/dns_records?page=1&per_page=50", []DnsRecord{
			{"CNAME", "www.example.com", "other-host.com"},
		})

		resolver := newCloudflareResolverWithStubs()
		// Combine the two responses into a single map
		resolver.c = &ClientStub{responses: combineResponses(stubbedZoneResponses, stubbedDnsResponses)}
		// Defer to the parent resolver for the ip of the cname
		resolver.parent = &IpResolverStub{ips: map[string]string{
			"other-host.com": "127.0.0.8",
		}}

		got, _ := resolver.Resolve("www.example.com")

		assert.DeepEqual(t, got, []string{"127.0.0.8"})
	})

	t.Run("it should return nil when any http error is encountered", func(t *testing.T) {
		resolver := newCloudflareResolverWithStubs()
		resolver.c = &ClientStub{errStub: errors.New("http error")}

		got, _ := resolver.Resolve("example.com")

		assert.Assert(t, got == nil)
	})

	t.Run("it should not attempt an api request if the credentials dont match the nameservers", func(t *testing.T) {
	})

	t.Run("it should check the base domains nameservers when resolving a subdomain", func(t *testing.T) {
		var tests = []struct {
			name   string
			domain string
			want   bool
		}{
			{"subdomain with base domain behind cloudflare", "sub.domain-behind-cloudflare.com", true},
			{"subdomain with base domain not behind cloudflare", "sub.example.com", false},
			{"subdomain with 2 dots in TLD", "sub.example.co.uk", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				resolver := newCloudflareResolverWithStubs()

				got := resolver.IsBehindCloudFlare(tt.domain)

				assert.DeepEqual(t, got, tt.want)
			})
		}
	})

	t.Run("it should cache nameserver lookups for the base domain", func(t *testing.T) {
		spyCalls := 0
		resolver := newCloudflareResolverWithStubs()
		resolver.lookupNs = func(host string) ([]*net.NS, error) {
			spyCalls++
			return NsLookupStub(host)
		}

		resolver.IsBehindCloudFlare("example.com")
		resolver.IsBehindCloudFlare("sub.example.com")

		assert.Equal(t, spyCalls, 1)
	})
}

func makeStubbedZoneResponse(endpoint string, zones []Zone) map[string]string {
	response := CloudflareResponse[[]Zone]{
		ResultInfo: ResultInfo{
			Page:       1,
			PerPage:    50,
			TotalPages: 1,
			Count:      1,
			TotalCount: 1,
		},
		Result:   zones,
		Success:  true,
		Errors:   nil,
		Messages: nil,
	}

	j, _ := json.Marshal(response)

	return map[string]string{endpoint: string(j)}
}

func makeStubbedDnsResponse(endpoint string, records []DnsRecord) map[string]string {
	response := CloudflareResponse[[]DnsRecord]{
		ResultInfo: ResultInfo{
			Page:       1,
			PerPage:    50,
			TotalPages: 1,
			Count:      1,
			TotalCount: 1,
		},
		Result:   records,
		Success:  true,
		Errors:   nil,
		Messages: nil,
	}

	j, _ := json.Marshal(response)

	return map[string]string{endpoint: string(j)}
}

func newCloudflareResolverWithStubs() *CloudflareResolver {
	return NewCloudflareResolver(
		log.New(io.Discard, "", 0),
		&IpResolverStub{},
		&ClientStub{},
		&Credentials{"foo", "bar"},
		[]string{"foo.ns.cloudflare.com", "bar.ns.cloudflare.com"},
		NsLookupStub,
	)
}

func combineResponses(responses ...map[string]string) map[string]string {
	var m = make(map[string]string)
	for _, r := range responses {
		for k, v := range r {
			m[k] = v
		}
	}

	return m
}

type ClientStub struct {
	responses map[string]string
	errStub   error
	calls     int
}

func (c *ClientStub) GetFromCacheOrFetchWithRateLimit(req http.Request) (string, error) {
	c.calls++
	return c.responses[req.Url], c.errStub
}
