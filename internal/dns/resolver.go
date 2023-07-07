package dns

import (
	"errors"
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/http"
	"log"
	"net"
)

var (
	ErrorDomainBehindCloudFlare = errors.New("domain is behind CloudFlare")
)

type cloudflareResolver interface {
	IpResolver
	IsBehindCloudFlare(domain string) bool
}

type Resolver struct {
	cfResolver cloudflareResolver
	lookupIp   IpLookupFunc
	l          *log.Logger
}

type IpLookupFunc func(host string) ([]net.IP, error)
type NsLookupFunc func(host string) ([]*net.NS, error)

func NewResolver(cfResolver cloudflareResolver, ipLookup IpLookupFunc, l *log.Logger) *Resolver {
	// Default to net.LookupIP
	if ipLookup == nil {
		ipLookup = net.LookupIP
	}

	resolver := &Resolver{cfResolver: cfResolver, lookupIp: ipLookup, l: l}

	if cfResolver == nil {
		resolver.cfResolver = NewCloudflareResolver(
			l,
			resolver,
			http.NewClient(l),
			nil,
			nil,
			net.LookupNS,
		)
	}

	return resolver
}

func (r *Resolver) Resolve(domain string) ([]string, error) {
	// If the domain is behind CloudFlare, we won't be able to resolve the real IP addresses unless
	// we have CloudFlare API credentials for the domain.
	if r.cfResolver.IsBehindCloudFlare(domain) {
		resolved, err := r.cfResolver.Resolve(domain)
		if err != nil {
			return nil, fmt.Errorf("%w: cloudflare error", ErrorDomainBehindCloudFlare)
		}
		return resolved, nil
	}

	r.l.Println("Looking up IP addresses for", domain, "...")
	ips, _ := r.lookupIp(domain)
	r.l.Println("IP addresses for", domain, "are", ips)

	var ipStrings []string

	for _, ip := range ips {
		ipStrings = append(ipStrings, ip.String())
	}

	return ipStrings, nil
}
