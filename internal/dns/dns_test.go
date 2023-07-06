package dns

import (
	"errors"
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
}

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
	return []string{ip}, nil
}
