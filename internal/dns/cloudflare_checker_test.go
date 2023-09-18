package dns

import (
	"gotest.tools/v3/assert"
	"io"
	"log"
	"net"
	"testing"
)

func TestCloudflareCredentialsChecker(t *testing.T) {
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
				checker := newCloudflareCredentialsCheckerWithStubs()

				got := checker.IsBehindCloudFlare(tt.domain)

				assert.DeepEqual(t, got, tt.want)
			})
		}
	})

	t.Run("it should cache nameserver lookups for the base domain", func(t *testing.T) {
		spyCalls := 0
		checker := newCloudflareCredentialsCheckerWithStubs()
		checker.lookupNs = func(host string) ([]*net.NS, error) {
			spyCalls++
			return NsLookupStub(host)
		}

		checker.IsBehindCloudFlare("example.com")
		checker.IsBehindCloudFlare("sub.example.com")

		assert.Equal(t, spyCalls, 1)
	})

	t.Run("it should return an error if there is a problem looking up nameservers for a domain", func(t *testing.T) {
	})

	t.Run("it should prompt for api credentials for each unique cloudflare nameservers", func(t *testing.T) {
		var tests = []struct {
			name       string
			domains    []UnresolvedDomain
			responses  []ExpectedResponse
			wantResult []UnresolvedDomain
		}{
			{
				name: "1 domain simple happy path",
				domains: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
				},
				responses: []ExpectedResponse{
					{"Detected 1", "y"},
					{"enter credentials for bar.ns.cloudflare.com, foo.ns.cloudflare.com (domains: domain-behind-cloudflare.com)", "y"},
					{"Email:", "foo@example.com"},
					{"API Token:", "1234567890"},
				},
				wantResult: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
							CloudflareCredentials: &Credentials{"foo@example.com", "1234567890"},
						},
					},
				},
			},
			{
				name: "1 domain decline first prompt",
				domains: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
				},
				responses: []ExpectedResponse{
					{"Detected 1", "n"},
				},
				wantResult: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
				},
			},
			{
				name: "1 domain decline first prompt (capital N)",
				domains: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
				},
				responses: []ExpectedResponse{
					{"Detected 1", "N"},
				},
				wantResult: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
				},
			},
			{
				name: "1 domain decline second prompt",
				domains: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
				},
				responses: []ExpectedResponse{
					{"Detected 1", "y"},
					{"enter credentials for bar.ns.cloudflare.com, foo.ns.cloudflare.com (domains: domain-behind-cloudflare.com)", "n"},
				},
				wantResult: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
				},
			},
			{
				name: "1 domain decline second prompt (capital N)",
				domains: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
				},
				responses: []ExpectedResponse{
					{"Detected 1", "Y"},
					{"enter credentials for bar.ns.cloudflare.com, foo.ns.cloudflare.com (domains: domain-behind-cloudflare.com)", "N"},
				},
				wantResult: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
				},
			},
			{
				name: "1 domain behind cloudflare, 1 domain not",
				domains: []UnresolvedDomain{
					{
						Name: "example.com",
					},
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
				},
				responses: []ExpectedResponse{
					{"Detected 1", "y"},
					{"enter credentials for bar.ns.cloudflare.com, foo.ns.cloudflare.com (domains: domain-behind-cloudflare.com)", "y"},
					{"Email:", "foo@example.com"},
					{"API Token:", "1234567890"},
				},
				wantResult: []UnresolvedDomain{
					{
						Name: "example.com",
					},
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
							CloudflareCredentials: &Credentials{"foo@example.com", "1234567890"},
						},
					},
				},
			},
			{
				name: "2 domains 1 cloudflare account",
				domains: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
					{
						Name: "sub.domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
				},
				responses: []ExpectedResponse{
					{"Detected 1", "y"},
					{"enter credentials for bar.ns.cloudflare.com, foo.ns.cloudflare.com (domains: domain-behind-cloudflare.com, sub.domain-behind-cloudflare.com)", "y"},
					{"Email:", "foo@example.com"},
					{"API Token:", "1234567890"},
				},
				wantResult: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
							CloudflareCredentials: &Credentials{"foo@example.com", "1234567890"},
						},
					},
					{
						Name: "sub.domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
							CloudflareCredentials: &Credentials{"foo@example.com", "1234567890"},
						},
					},
				},
			},
			{
				name: "2 domains 2 cloudflare accounts",
				domains: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
					{
						Name: "another-domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"baz.ns.cloudflare.com", "bing.ns.cloudflare.com"},
						},
					},
				},
				responses: []ExpectedResponse{
					{"Detected 2", "y"},
					{"enter credentials for bar.ns.cloudflare.com, foo.ns.cloudflare.com (domains: domain-behind-cloudflare.com)", "y"},
					{"Email:", "foo@example.com"},
					{"API Token:", "1234567890"},
					{"enter credentials for baz.ns.cloudflare.com, bing.ns.cloudflare.com (domains: another-domain-behind-cloudflare.com)", "y"},
					{"Email:", "bar@example.com"},
					{"API Token:", "9876543210"},
				},
				wantResult: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
							CloudflareCredentials: &Credentials{"foo@example.com", "1234567890"},
						},
					},
					{
						Name: "another-domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"baz.ns.cloudflare.com", "bing.ns.cloudflare.com"},
							CloudflareCredentials: &Credentials{"bar@example.com", "9876543210"},
						},
					},
				},
			},
			{
				name: "2 domains 2 cloudflare accounts but only 1 credentials",
				domains: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
						},
					},
					{
						Name: "another-domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"baz.ns.cloudflare.com", "bing.ns.cloudflare.com"},
						},
					},
				},
				responses: []ExpectedResponse{
					{"Detected 2", "y"},
					{"enter credentials for bar.ns.cloudflare.com, foo.ns.cloudflare.com (domains: domain-behind-cloudflare.com)", "y"},
					{"Email:", "foo@example.com"},
					{"API Token:", "1234567890"},
					{"enter credentials for baz.ns.cloudflare.com, bing.ns.cloudflare.com (domains: another-domain-behind-cloudflare.com)", "n"},
				},
				wantResult: []UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
							CloudflareCredentials: &Credentials{"foo@example.com", "1234567890"},
						},
					},
					{
						Name: "another-domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							BaseDomainNameservers: []string{"baz.ns.cloudflare.com", "bing.ns.cloudflare.com"},
							CloudflareCredentials: nil,
						},
					},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				spy := &SpyPrompter{StubbedResponses: tt.responses}

				checker := newCloudflareCredentialsCheckerWithStubs()
				checker.p = spy

				got := checker.PromptForCredentials(tt.domains)

				assert.DeepEqual(t, got, tt.wantResult)

				// Assert that the prompter was called with the correct messages
				assert.Equal(t, len(spy.Calls), len(tt.responses))
				for i := 0; i < len(tt.responses); i++ {
					assertStringContains(t, spy.Calls[i], tt.responses[i].Prompt)
				}
			})
		}
	})
}

type ExpectedResponse struct {
	Prompt string
	Answer string
}

func newCloudflareCredentialsCheckerWithStubs() *CloudflareCredentialsChecker {
	return NewCloudflareCredentialsChecker(
		log.New(io.Discard, "", 0),
		&SpyPrompter{},
		NsLookupStub,
	)
}

type SpyPrompter struct {
	StubbedResponses []ExpectedResponse
	Calls            []string
}

func (s *SpyPrompter) Prompt(msg, defaultResponse string, validResponses []string) string {
	i := len(s.Calls)
	s.Calls = append(s.Calls, msg)
	return s.StubbedResponses[i].Answer
}
