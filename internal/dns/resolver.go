package dns

import (
	"golang.org/x/net/publicsuffix"
	"log"
	"net"
	"strings"
)

type Resolver struct {
	cfResolver IpResolver
	lookupIp   IpLookupFunc
	lookupNs   NsLookupFunc
	l          *log.Logger
	cachedNs   map[string][]string
}

type IpLookupFunc func(host string) ([]net.IP, error)
type NsLookupFunc func(host string) ([]*net.NS, error)

func NewResolver(cfResolver IpResolver, ipLookup IpLookupFunc, nsLookup NsLookupFunc, l *log.Logger) *Resolver {
	// Default to net.LookupIP
	if ipLookup == nil {
		ipLookup = net.LookupIP
	}

	// Default to net.LookupNS
	if nsLookup == nil {
		nsLookup = net.LookupNS
	}

	return &Resolver{cfResolver: cfResolver, lookupIp: ipLookup, lookupNs: nsLookup, l: l}
}

func (r *Resolver) Resolve(domain string) ([]string, error) {
	// If the domain is behind CloudFlare, we won't be able to resolve the real IP addresses unless
	// we have CloudFlare API credentials for the domain.
	if r.isBehindCloudFlare(getBaseDomain(domain)) {
		return r.cfResolver.Resolve(domain)
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

func (r *Resolver) isBehindCloudFlare(domain string) bool {
	// Check if we've already looked up the nameservers for this domain
	if r.cachedNs == nil || r.cachedNs[domain] == nil {
		r.l.Println("Looking up nameservers for", domain, "...")
		ns, _ := r.lookupNs(domain)
		var nsStrings []string
		for _, n := range ns {
			nsStrings = append(nsStrings, n.Host)
		}
		r.l.Println("Nameservers for", domain, "are", nsStrings)

		// Cache the nameservers for this domain
		r.cachedNs = map[string][]string{domain: nsStrings}
	}

	ns := r.cachedNs[domain]

	for _, n := range ns {
		// Check if the nameserver format matches *.ns.cloudflare.com.
		if len(n) >= 18 && n[len(n)-18:] == "ns.cloudflare.com." {
			return true
		}
	}

	return false
}

func getBaseDomain(domain string) string {
	// Get the effective top level domain (eTLD)
	eTLD, _ := publicsuffix.PublicSuffix(domain)

	// Get the domain without the eTLD
	domainWithoutTld := strings.TrimSuffix(domain, "."+eTLD)

	// Split the domain by the dot character
	parts := strings.Split(domainWithoutTld, ".")

	// The last part is the base domain
	return parts[len(parts)-1] + "." + eTLD
}
