package dns

const (
	OK int = iota
	STRANDED
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

	return STRANDED
}

// IpResolver is an interface for resolving a domain to its IP address(s). It will return
// the ip addresses when it can, or an error if it cannot.
type IpResolver interface {
	Resolve(domain string) ([]string, error)
}
