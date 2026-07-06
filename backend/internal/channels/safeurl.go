package channels

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrUnsafeOutboundURL is returned when a tenant-supplied URL points at a
// disallowed scheme or a non-public network target (loopback, private ranges,
// link-local, etc.). It guards outbound requests built from channel metadata
// against SSRF.
var ErrUnsafeOutboundURL = errors.New("unsafe outbound url")

// ValidateOutboundURL ensures a tenant-controlled URL is safe to call from the
// server. Only https is allowed, and the target must not resolve to loopback,
// private, link-local, unspecified, or multicast addresses.
func ValidateOutboundURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("%w: empty url", ErrUnsafeOutboundURL)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnsafeOutboundURL, err)
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("%w: only https is allowed", ErrUnsafeOutboundURL)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("%w: missing host", ErrUnsafeOutboundURL)
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") {
		return fmt.Errorf("%w: localhost is not allowed", ErrUnsafeOutboundURL)
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("%w: blocked ip %s", ErrUnsafeOutboundURL, ip)
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("%w: cannot resolve host: %v", ErrUnsafeOutboundURL, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("%w: host did not resolve", ErrUnsafeOutboundURL)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("%w: host resolves to blocked ip %s", ErrUnsafeOutboundURL, ip)
		}
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast()
}
