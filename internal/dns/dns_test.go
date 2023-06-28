package dns

import (
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
				"stranded",
				"stranded.example.com",
				"127.0.0.1",
				map[string]string{
					"stranded.example.com": "0.0.0.0",
				},
				STRANDED,
			},
			{
				"unknown",
				"unknown.example.com",
				"127.0.0.1",
				map[string]string{},
				UNKNOWN,
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
}

type IpResolverStub struct {
	ips map[string]string
}

func (s *IpResolverStub) Resolve(domain string) []string {
	ip, ok := s.ips[domain]
	if !ok {
		return nil
	}
	return []string{ip}
}
