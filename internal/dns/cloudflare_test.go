package dns

import (
	"encoding/json"
	"errors"
	"github.com/jfortunato/serverpilot-tools/internal/http"
	"gotest.tools/v3/assert"
	"io"
	"log"
	"testing"
)

func TestCloudflare(t *testing.T) {
	t.Run("it should not be able to resolve a domain behind CloudFlare nameservers that we don't have api credentials for", func(t *testing.T) {
		clientStub := &ClientStub{}

		resolver := NewCloudflareResolver(log.New(io.Discard, "", 0), clientStub, nil, nil)

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
		}

		// These tests will always use the same zone response
		stubbedZoneResponses := makeStubbedZoneResponse("https://api.cloudflare.com/client/v4/zones?page=1&per_page=50", []Zone{{"1"}})

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				stubbedDnsResponses := makeStubbedDnsResponse("https://api.cloudflare.com/client/v4/zones/1/dns_records?page=1&per_page=50", tt.records)

				// Combine the two responses into a single map
				clientStub := &ClientStub{responses: combineResponses(stubbedZoneResponses, stubbedDnsResponses)}

				resolver := NewCloudflareResolver(log.New(io.Discard, "", 0), clientStub, &Credentials{"foo", "bar"}, []string{"foo.ns.cloudflare.com", "bar.ns.cloudflare.com"})

				got, _ := resolver.Resolve(tt.domain)

				assert.DeepEqual(t, got, tt.want)
			})
		}
	})

	t.Run("it should return nil when any http error is encountered", func(t *testing.T) {
		clientStub := &ClientStub{errStub: errors.New("http error")}

		resolver := NewCloudflareResolver(log.New(io.Discard, "", 0), clientStub, &Credentials{"foo", "bar"}, []string{"foo.ns.cloudflare.com", "bar.ns.cloudflare.com"})

		got, _ := resolver.Resolve("example.com")

		assert.Assert(t, got == nil)
	})

	t.Run("it should cache the dns records in memory", func(t *testing.T) {
		// Combine the two responses into a single map
		clientStub := &ClientStub{responses: combineResponses(
			makeStubbedZoneResponse("https://api.cloudflare.com/client/v4/zones?page=1&per_page=50", []Zone{{"1"}}),
			makeStubbedDnsResponse("https://api.cloudflare.com/client/v4/zones/1/dns_records?page=1&per_page=50", []DnsRecord{{"A", "example.com", "127.0.0.1"}}),
		)}

		resolver := NewCloudflareResolver(log.New(io.Discard, "", 0), clientStub, &Credentials{"foo", "bar"}, []string{"foo.ns.cloudflare.com", "bar.ns.cloudflare.com"})

		_, _ = resolver.Resolve("example.com")
		_, _ = resolver.Resolve("example.com")

		// Each Resolve call should result in 2 api requests (one for the zone, one for the dns records)
		// The second call should not result in any api requests.
		assert.Equal(t, clientStub.calls, 2)
	})

	t.Run("it should not attempt an api request if the credentials dont match the nameservers", func(t *testing.T) {
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
