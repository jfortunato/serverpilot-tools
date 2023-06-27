package dns

import (
	"errors"
	"io"
	"log"
	"net"
	"reflect"
	"testing"
)

func TestResolver(t *testing.T) {
	t.Run("it should resolve the ip addresses for the given domain name", func(t *testing.T) {
		resolver := NewResolver(IpLookupStub, NsLookupStub, log.New(io.Discard, "", 0))

		got := resolver.Resolve("example.com")
		want := []string{"127.0.0.1"}

		assertEqualSlice(t, got, want)
	})

	t.Run("it should return an error that occurs during the network request", func(t *testing.T) {
	})

	t.Run("it should not be able to resolve a domain behind CloudFlare nameservers that we don't have api credentials for", func(t *testing.T) {
		resolver := NewResolver(IpLookupStub, NsLookupStub, log.New(io.Discard, "", 0))

		got := resolver.Resolve("domain-behind-cloudflare.com")

		if got != nil {
			t.Errorf("got %v want %v", got, nil)
		}
	})

	t.Run("it should check the base domains nameservers when resolving a subdomain", func(t *testing.T) {
		var tests = []struct {
			name   string
			domain string
			want   []string
		}{
			{"subdomain with base domain behind cloudflare", "sub.domain-behind-cloudflare.com", nil},
			{"subdomain with base domain not behind cloudflare", "sub.example.com", []string{"127.0.0.2"}},
			{"subdomain with 2 dots in TLD", "sub.example.co.uk", []string{"127.0.0.3"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				resolver := NewResolver(IpLookupStub, NsLookupStub, log.New(io.Discard, "", 0))

				got := resolver.Resolve(tt.domain)

				assertEqualSlice(t, got, tt.want)
			})
		}
	})
}

func IpLookupStub(host string) ([]net.IP, error) {
	known := map[string][]net.IP{
		"example.com":                      {net.ParseIP("127.0.0.1")},
		"sub.example.com":                  {net.ParseIP("127.0.0.2")},
		"sub.example.co.uk":                {net.ParseIP("127.0.0.3")},
		"domain-behind-cloudflare.com":     {net.ParseIP("1.1.1.1")},
		"sub.domain-behind-cloudflare.com": {net.ParseIP("1.0.0.1")},
	}

	if ips, ok := known[host]; ok {
		return ips, nil
	}

	return nil, nil
}

func NsLookupStub(host string) ([]*net.NS, error) {
	known := map[string][]*net.NS{
		"example.com":                  {&net.NS{Host: "ns1.example.com."}},
		"example.co.uk":                {&net.NS{Host: "ns1.example.co.uk."}},
		"domain-behind-cloudflare.com": {&net.NS{Host: "foo.ns.cloudflare.com."}, &net.NS{Host: "bar.ns.cloudflare.com."}},
	}

	if ns, ok := known[host]; ok {
		return ns, nil
	}

	// Don't allow any other hostnames to be resolved
	panic(errors.New("unknown host " + host))
}

func assertEqualSlice(t *testing.T, got, want any) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}
