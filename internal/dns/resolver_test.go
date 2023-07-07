package dns

import (
	"errors"
	"gotest.tools/v3/assert"
	"io"
	"log"
	"net"
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
			domain             string
			isBehindCloudflare bool
			stubbedResult      string
			expectedResult     []string
		}{
			{"domain-behind-cloudflare.com", true, "", nil},
			{"domain-behind-cloudflare.com", true, "127.0.0.1", []string{"127.0.0.1"}},
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
				resolver.cfResolver = &CloudflareResolverStub{IpResolverStub: &IpResolverStub{ips: stub}, isBehindCloudflare: tt.isBehindCloudflare}

				got, _ := resolver.Resolve(tt.domain)

				assert.DeepEqual(t, got, tt.expectedResult)
			})
		}
	})

	t.Run("it should return an error when the domain is behind cloudflare but the cloudflare resolver cannot resolve", func(t *testing.T) {
		resolver := newResolverWithStubs()
		resolver.cfResolver = &CloudflareResolverStub{&IpResolverStub{}, true}

		got, err := resolver.Resolve("domain-behind-cloudflare.com")

		assert.Assert(t, got == nil)
		assert.ErrorIs(t, err, ErrorDomainBehindCloudFlare)
	})
}

func newResolverWithStubs() *Resolver {
	return NewResolver(
		&CloudflareResolverStub{&IpResolverStub{}, false},
		IpLookupStub,
		log.New(io.Discard, "", 0),
	)
}

type CloudflareResolverStub struct {
	*IpResolverStub
	isBehindCloudflare bool
}

func (r *CloudflareResolverStub) IsBehindCloudFlare(domain string) bool {
	return r.isBehindCloudflare
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
