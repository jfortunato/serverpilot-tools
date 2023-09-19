package dns

import (
	"errors"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"gotest.tools/v3/assert"
	"io"
	"log"
	"testing"
)

func TestDNS(t *testing.T) {
	t.Run("it should return the correct status for the domain", func(t *testing.T) {
		var tests = []struct {
			name        string
			domain      string
			serverIp    string
			resolvedIps map[string]string
			want        int
		}{
			{
				"ok",
				"example.com",
				"127.0.0.1",
				map[string]string{
					"example.com": "127.0.0.1",
				},
				OK,
			},
			{
				"inactive",
				"inactive.example.com",
				"127.0.0.1",
				map[string]string{
					"inactive.example.com": "0.0.0.0",
				},
				INACTIVE,
			},
			{
				"unknown",
				"unknown.example.com",
				"127.0.0.1",
				nil,
				UNKNOWN,
			},
			{
				"expired/not pointed",
				"expired.com",
				"127.0.0.1",
				map[string]string{},
				INACTIVE,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				checker := NewDnsChecker(&IpResolverStub{tt.resolvedIps}, nil)

				got := checker.CheckStatus(UnresolvedDomain{Name: tt.domain}, tt.serverIp)
				want := tt.want

				assert.Equal(t, got, want)
			})
		}
	})

	t.Run("it should return a list of inactive app domains", func(t *testing.T) {
		var tests = []struct {
			name           string
			domains        []UnresolvedDomain
			appservers     []serverpilot.AppServer
			resolvedIps    map[string]string
			includeUnknown bool
			want           []AppDomainStatus
		}{
			{
				"only inactive domains",
				[]UnresolvedDomain{
					{Name: "ok.example.com"},
					{Name: "inactive.example.com"},
					{Name: "unknown.example.com"},
				},
				[]serverpilot.AppServer{
					{serverpilot.App{Id: "1", Domains: []string{"ok.example.com"}}, serverpilot.Server{Name: "server1", Ipaddress: "127.0.0.1"}},
					{serverpilot.App{Id: "2", Domains: []string{"inactive.example.com"}}, serverpilot.Server{Name: "server1", Ipaddress: "127.0.0.1"}},
					{serverpilot.App{Id: "3", Domains: []string{"unknown.example.com"}}, serverpilot.Server{Name: "server1", Ipaddress: "127.0.0.1"}},
				},
				map[string]string{
					"ok.example.com":       "127.0.0.1",
					"inactive.example.com": "0.0.0.0",
					"unknown.example.com":  "unknown",
				},
				false,
				[]AppDomainStatus{
					{AppId: "2", Domain: "inactive.example.com", ServerName: "server1", Status: INACTIVE},
				},
			},
			{
				"inactive & unknown domains",
				[]UnresolvedDomain{
					{Name: "ok.example.com"},
					{Name: "inactive.example.com"},
					{Name: "unknown.example.com"},
				},
				[]serverpilot.AppServer{
					{serverpilot.App{Id: "1", Domains: []string{"ok.example.com"}}, serverpilot.Server{Name: "server1", Ipaddress: "127.0.0.1"}},
					{serverpilot.App{Id: "2", Domains: []string{"inactive.example.com"}}, serverpilot.Server{Name: "server1", Ipaddress: "127.0.0.1"}},
					{serverpilot.App{Id: "3", Domains: []string{"unknown.example.com"}}, serverpilot.Server{Name: "server1", Ipaddress: "127.0.0.1"}},
				},
				map[string]string{
					"ok.example.com":       "127.0.0.1",
					"inactive.example.com": "0.0.0.0",
					"unknown.example.com":  "unknown",
				},
				true,
				[]AppDomainStatus{
					{AppId: "2", Domain: "inactive.example.com", ServerName: "server1", Status: INACTIVE},
					{AppId: "3", Domain: "unknown.example.com", ServerName: "server1", Status: UNKNOWN},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				checker := NewDnsChecker(&IpResolverStub{tt.resolvedIps}, nil)

				got := checker.GetInactiveAppDomains(&FakeTicker{}, tt.domains, tt.appservers, tt.includeUnknown)
				want := tt.want

				assert.DeepEqual(t, got, want)
			})
		}
	})

	t.Run("it should evaluate domains", func(t *testing.T) {
		var tests = []struct {
			name    string
			domains []string
			want    []UnresolvedDomain
		}{
			{
				"1 simple domain, not behind cloudflare",
				[]string{"example.com"},
				[]UnresolvedDomain{
					{
						Name: "example.com",
					},
				},
			},
			{
				"1 subdomain, not behind cloudflare",
				[]string{"sub.example.com"},
				[]UnresolvedDomain{
					{
						Name: "sub.example.com",
					},
				},
			},
			{
				"1 simple domain behind cloudflare",
				[]string{"domain-behind-cloudflare.com"},
				[]UnresolvedDomain{
					{
						Name: "domain-behind-cloudflare.com",
						CloudflareMetadata: &CloudflareDomainMetadata{
							[]string{"bar.ns.cloudflare.com", "foo.ns.cloudflare.com"},
							nil,
						},
					},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				checker := NewDnsChecker(nil, &CloudflareCredentialsChecker{
					l:        log.New(io.Discard, "", 0),
					p:        nil,
					lookupNs: NsLookupStub,
					cachedNs: nil,
				})

				got := checker.EvaluateDomains(&FakeTicker{}, tt.domains)

				assert.DeepEqual(t, got, tt.want)
			})
		}
	})
}

type FakeTicker struct{}

func (t *FakeTicker) Tick() {}

type IpResolverStub struct {
	ips map[string]string
}

func (s *IpResolverStub) Resolve(domain UnresolvedDomain) ([]string, error) {
	if s.ips == nil {
		return nil, errors.New("no ips")
	}

	ip, ok := s.ips[domain.Name]
	if !ok {
		return nil, nil
	}
	// We needed a way to differentiate between an inactive domain and an unknown domain
	if ip == "unknown" {
		return nil, errors.New("unknown")
	}
	return []string{ip}, nil
}
