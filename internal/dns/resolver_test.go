package dns

import (
	"errors"
	"gotest.tools/v3/assert"
	"io"
	"log"
	"net"
	"strings"
	"testing"
)

func TestResolver(t *testing.T) {
	t.Run("it should resolve the ip addresses for the given domain name", func(t *testing.T) {
		resolver := newResolverWithStubs()

		got, _ := resolver.Resolve("example.com")
		want := []string{"127.0.0.1"}

		assert.DeepEqual(t, got, want)
	})

	t.Run("it should return an error that occurs during the network request", func(t *testing.T) {
	})

	t.Run("it should defer to the cloudflare resolver when the domain is on cloudflare nameservers", func(t *testing.T) {
		var tests = []struct {
			domain         string
			stubbedResult  string
			expectedResult []string
		}{
			{"domain-behind-cloudflare.com", "", nil},
			{"domain-behind-cloudflare.com", "127.0.0.1", []string{"127.0.0.1"}},
		}

		for _, tt := range tests {
			t.Run("it returns the result from the cloudflare resolver", func(t *testing.T) {
				resolver := newResolverWithStubs()

				// Override the cloudflare resolver with this stub
				var stub map[string]string
				if tt.stubbedResult == "" {
					stub = map[string]string{}
				} else {
					stub = map[string]string{tt.domain: tt.stubbedResult}
				}
				resolver.cfResolver = &IpResolverStub{ips: stub}

				got, _ := resolver.Resolve(tt.domain)

				assert.DeepEqual(t, got, tt.expectedResult)
			})
		}
	})

	t.Run("it should return an error when the domain is behind cloudflare but the cloudflare resolver cannot resolve", func(t *testing.T) {
		resolver := newResolverWithStubs()
		resolver.cfResolver = &IpResolverStub{}

		got, err := resolver.Resolve("domain-behind-cloudflare.com")

		assert.Assert(t, got == nil)
		assert.ErrorIs(t, err, ErrorDomainBehindCloudFlare)
	})
}

func newResolverWithStubs() *Resolver {
	return NewResolver(
		&IpResolverStub{},
		&CloudflareCheckerStub{},
		nil,
		IpLookupStub,
		log.New(io.Discard, "", 0),
	)
}

type CloudflareCheckerStub struct {
}

func (c *CloudflareCheckerStub) IsBehindCloudFlare(domain string) bool {
	ns, _ := c.GetNameserversForBaseDomain(domain)

	return len(ns) > 0 && strings.HasSuffix(ns[0], "ns.cloudflare.com")
}

func (c *CloudflareCheckerStub) GetNameserversForBaseDomain(domain string) ([]string, error) {
	ns, _ := NsLookupStub(getBaseDomain(domain))
	var nameservers []string
	for _, n := range ns {
		// Remove trailing dot
		host := strings.TrimSuffix(n.Host, ".")

		nameservers = append(nameservers, host)
	}

	return nameservers, nil
}

func IpLookupStub(host string) ([]net.IP, error) {
	known := map[string][]net.IP{
		"example.com":                          {net.ParseIP("127.0.0.1")},
		"sub.example.com":                      {net.ParseIP("127.0.0.2")},
		"sub.example.co.uk":                    {net.ParseIP("127.0.0.3")},
		"domain-behind-cloudflare.com":         {net.ParseIP("1.1.1.1")},
		"sub.domain-behind-cloudflare.com":     {net.ParseIP("1.0.0.1")},
		"another-domain-behind-cloudflare.com": {net.ParseIP("1.1.1.1")},
	}

	if ips, ok := known[host]; ok {
		return ips, nil
	}

	return nil, nil
}

func NsLookupStub(host string) ([]*net.NS, error) {
	known := map[string][]*net.NS{
		"example.com":                          {&net.NS{Host: "ns1.example.com."}},
		"example.co.uk":                        {&net.NS{Host: "ns1.example.co.uk."}},
		"domain-behind-cloudflare.com":         {&net.NS{Host: "foo.ns.cloudflare.com."}, &net.NS{Host: "bar.ns.cloudflare.com."}},
		"another-domain-behind-cloudflare.com": {&net.NS{Host: "baz.ns.cloudflare.com."}, &net.NS{Host: "bing.ns.cloudflare.com."}},
	}

	if ns, ok := known[host]; ok {
		return ns, nil
	}

	// Don't allow any other hostnames to be resolved
	panic(errors.New("unknown host " + host))
}
