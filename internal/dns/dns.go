package dns

import (
	"golang.org/x/net/publicsuffix"
	"strings"
)

const (
	OK int = iota
	INACTIVE
	UNKNOWN
)

type DnsChecker struct {
	r IpResolver
}

func NewDnsChecker(r IpResolver) *DnsChecker {
	return &DnsChecker{r}
}

func (c *DnsChecker) CheckStatus(domain string, serverIp string) int {
	resolvedIps, err := c.r.Resolve(domain)
	if err != nil {
		return UNKNOWN
	}

	for _, ip := range resolvedIps {
		if ip == serverIp {
			return OK
		}
	}

	return INACTIVE
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

// IpResolver is an interface for resolving a domain to its IP address(s). It will return
// the ip addresses when it can, or an error if it cannot.
type IpResolver interface {
	Resolve(domain string) ([]string, error)
}
