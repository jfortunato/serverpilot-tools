package dns

import (
	"golang.org/x/net/publicsuffix"
	"log"
	"net"
	"strings"
)

type Resolver struct {
	lookupIp IpLookupFunc
	lookupNs NsLookupFunc
	l        *log.Logger
}

type IpLookupFunc func(host string) ([]net.IP, error)
type NsLookupFunc func(host string) ([]*net.NS, error)

func NewResolver(ipLookup IpLookupFunc, nsLookup NsLookupFunc, l *log.Logger) *Resolver {
	// Default to net.LookupIP
	if ipLookup == nil {
		ipLookup = net.LookupIP
	}

	// Default to net.LookupNS
	if nsLookup == nil {
		nsLookup = net.LookupNS
	}

	return &Resolver{ipLookup, nsLookup, l}
}

func (r *Resolver) Resolve(domain string) []string {
	// If the domain is behind CloudFlare, we won't be able to resolve the real IP addresses unless
	// we have CloudFlare API credentials for the domain.
	if r.isBehindCloudFlare(getBaseDomain(domain)) {
		return nil
	}

	r.l.Println("Looking up IP addresses for", domain, "...")
	ips, _ := r.lookupIp(domain)
	r.l.Println("IP addresses for", domain, "are", ips)

	var ipStrings []string

	for _, ip := range ips {
		ipStrings = append(ipStrings, ip.String())
	}

	return ipStrings
}

func (r *Resolver) isBehindCloudFlare(domain string) bool {
	r.l.Println("Looking up nameservers for", domain, "...")
	ns, _ := r.lookupNs(domain)
	var nsStrings []string
	for _, n := range ns {
		nsStrings = append(nsStrings, n.Host)
	}
	r.l.Println("Nameservers for", domain, "are", nsStrings)

	for _, n := range ns {
		// Check if the nameserver format matches *.ns.cloudflare.com.
		if len(n.Host) >= 18 && n.Host[len(n.Host)-18:] == "ns.cloudflare.com." {
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
