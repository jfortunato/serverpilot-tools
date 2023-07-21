package dns

import (
	"errors"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"gotest.tools/v3/assert"
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
				checker := NewDnsChecker(&IpResolverStub{tt.resolvedIps})

				got := checker.CheckStatus(tt.domain, tt.serverIp)
				want := tt.want

				assert.Equal(t, got, want)
			})
		}
	})

	t.Run("it should return a list of inactive app domains", func(t *testing.T) {
		var tests = []struct {
			name           string
			appservers     []serverpilot.AppServer
			resolvedIps    map[string]string
			includeUnknown bool
			want           []AppDomainStatus
		}{
			{
				"only inactive domains",
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
				checker := NewDnsChecker(&IpResolverStub{tt.resolvedIps})

				got := checker.GetInactiveAppDomains(&FakeTicker{}, tt.appservers, tt.includeUnknown)
				want := tt.want

				assert.DeepEqual(t, got, want)
			})
		}
	})
}

type FakeTicker struct{}

func (t *FakeTicker) Tick() {}

type IpResolverStub struct {
	ips map[string]string
}

func (s *IpResolverStub) Resolve(domain string) ([]string, error) {
	if s.ips == nil {
		return nil, errors.New("no ips")
	}

	ip, ok := s.ips[domain]
	if !ok {
		return nil, nil
	}
	// We needed a way to differentiate between an inactive domain and an unknown domain
	if ip == "unknown" {
		return nil, errors.New("unknown")
	}
	return []string{ip}, nil
}
