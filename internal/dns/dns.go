package dns

const (
	OK int = iota
	STRANDED
	UNKNOWN
)

type DnsChecker struct {
	r   IpResolver
	ips []string
}

func NewDnsChecker(r IpResolver, serverIps []string) *DnsChecker {
	return &DnsChecker{r, serverIps}
}

func (c *DnsChecker) CheckStatus(domain string) int {
	resolvedIps := c.r.Resolve(domain)

	if resolvedIps == nil {
		return UNKNOWN
	}

	for _, ip := range resolvedIps {
		for _, serverIp := range c.ips {
			if ip == serverIp {
				return OK
			}
		}
	}

	return STRANDED
}

// IpResolver is an interface for resolving a domain to its IP address(s). It will return
// the ip addresses when it can, or a nil slice if it cannot.
type IpResolver interface {
	Resolve(domain string) []string
}
