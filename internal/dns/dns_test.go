package dns

import "testing"

func TestDNS(t *testing.T) {
	t.Run("it should return the correct status for the domain", func(t *testing.T) {
		var tests = []struct {
			name        string
			domain      string
			serverIps   []string
			resolvedIps map[string]string
			want        int
		}{
			{
				"ok",
				"example.com",
				[]string{"127.0.0.1"},
				map[string]string{
					"example.com": "127.0.0.1",
				},
				OK,
			},
			{
				"stranded",
				"stranded.example.com",
				[]string{"127.0.0.1"},
				map[string]string{
					"stranded.example.com": "0.0.0.0",
				},
				STRANDED,
			},
			{
				"unknown",
				"unknown.example.com",
				[]string{"127.0.0.1"},
				map[string]string{},
				UNKNOWN,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				checker := NewDnsChecker(&IpResolverStub{tt.resolvedIps}, tt.serverIps)

				got := checker.CheckStatus(tt.domain)
				want := tt.want

				if got != want {
					t.Errorf("got %q want %q", got, want)
				}
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
