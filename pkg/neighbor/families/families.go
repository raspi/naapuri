package families

type AddressFamily uint8

const (
	UnknownAddressFamily AddressFamily = iota
	IPv4                               // https://tools.ietf.org/html/rfc791
	IPv6                               // https://tools.ietf.org/html/rfc2460
)

func (af AddressFamily) String() string {
	switch af {
	case IPv4:
		return `ipv4`
	case IPv6:
		return `ipv6`
	default:
		return `???`
	}
}
