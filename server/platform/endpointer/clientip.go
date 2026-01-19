package endpointer

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/realclientip/realclientip-go"
)

// ClientIPStrategy extracts the real client IP from HTTP requests.
// This interface is compatible with realclientip.Strategy.
type ClientIPStrategy interface {
	ClientIP(headers http.Header, remoteAddr string) string
}

// singleIPHeaderNames are header names that contain a single IP address,
// typically set by CDNs or reverse proxies.
var singleIPHeaderNames = map[string]struct{}{
	"true-client-ip":   {}, // Cloudflare Enterprise, Akamai
	"x-real-ip":        {}, // Nginx
	"cf-connecting-ip": {}, // Cloudflare
	"x-azure-clientip": {}, // Azure
	"fastly-client-ip": {}, // Fastly
}

// NewClientIPStrategy creates a ClientIPStrategy based on the trusted_proxies configuration.
//
// Config values:
//   - "" (empty): Legacy behavior for backwards compatibility - trusts True-Client-IP,
//     X-Real-IP, and leftmost X-Forwarded-For. This is deprecated; use "none" when
//     exposing the server directly to the internet.
//   - "none": Ignores all headers, uses only RemoteAddr.
//   - A header name (e.g., "True-Client-IP", "X-Real-IP", "CF-Connecting-IP"):
//     Trust this single-IP header, fall back to RemoteAddr.
//   - A number (e.g., "2"): Trust X-Forwarded-For with this many proxy hops
//   - Comma-separated IPs/CIDRs (e.g., "10.0.0.0/8,192.168.0.0/16"):
//     Trust X-Forwarded-For from requests originating from these proxy ranges.
func NewClientIPStrategy(trustedProxies string) (ClientIPStrategy, error) {
	trustedProxies = strings.TrimSpace(trustedProxies)

	var strategy ClientIPStrategy
	var err error

	if trustedProxies == "" {
		// Empty: legacy behavior for backwards compatibility.
		strategy = &legacyStrategy{}
	} else if strings.EqualFold(trustedProxies, "none") {
		// "none": Trust no one; return (non-spoofable) RemoteAddr only.
		strategy = realclientip.RemoteAddrStrategy{}
	} else if _, ok := singleIPHeaderNames[strings.ToLower(trustedProxies)]; ok {
		// Check if it's a known single-IP header name.
		strategy, err = realclientip.NewSingleIPHeaderStrategy(trustedProxies)
		if err != nil {
			return nil, fmt.Errorf("invalid header name %q: %w", trustedProxies, err)
		}
	} else if hopCount, err := strconv.Atoi(trustedProxies); err == nil {
		// Check if it's a number (hop count).
		if hopCount < 1 {
			return nil, fmt.Errorf("trusted_proxies hop count must be >= 1, got %d", hopCount)
		}
		strategy, err = realclientip.NewRightmostTrustedCountStrategy("X-Forwarded-For", hopCount)
		if err != nil {
			return nil, fmt.Errorf("failed to create hop count strategy: %w", err)
		}
	} else {
		// Otherwise, parse as comma-separated IP ranges.
		rangeStrs := strings.Split(trustedProxies, ",")
		for i := range rangeStrs {
			rangeStrs[i] = strings.TrimSpace(rangeStrs[i])
		}

		trustedRanges, err := realclientip.AddressesAndRangesToIPNets(rangeStrs...)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted_proxies IP ranges: %w", err)
		}

		strategy, err = realclientip.NewRightmostTrustedRangeStrategy("X-Forwarded-For", trustedRanges)
		if err != nil {
			return nil, fmt.Errorf("failed to create IP range strategy: %w", err)
		}
	}

	// Chain strategy with RemoteAddr as fallback.
	return realclientip.NewChainStrategy(strategy, realclientip.RemoteAddrStrategy{}), nil
}

// legacyStrategy implements the original ExtractIP behavior for backwards compatibility.
// This is deprecated; if your server is exposed directly to the internet, switch to
// the "none" strategy.
type legacyStrategy struct{}

func (s *legacyStrategy) ClientIP(headers http.Header, remoteAddr string) string {
	// Build a minimal http.Request to pass to ExtractIP
	r := &http.Request{
		Header:     headers,
		RemoteAddr: remoteAddr,
	}
	return ExtractIP(r)
}
