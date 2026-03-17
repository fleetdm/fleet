package http

import (
	"errors"
	"net/url"
	"strings"
)

// MaskSecretURLParams masks URL query values if the query param name includes "secret", "token",
// "key", "password". It accepts a raw string and returns a redacted string if the raw string is
// URL-parseable. If it is not URL-parseable, the raw string is returned unchanged.
func MaskSecretURLParams(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	keywords := []string{"secret", "token", "key", "password"}
	containsKeyword := func(s string) bool {
		s = strings.ToLower(s)
		for _, kw := range keywords {
			if strings.Contains(s, kw) {
				return true
			}
		}
		return false
	}

	q := u.Query()
	for k := range q {
		if containsKeyword(k) {
			q[k] = []string{"MASKED"}
		}
	}
	u.RawQuery = q.Encode()

	return u.Redacted()
}

// MaskURLError checks if the provided error is a *url.Error. If so, it applies MaskSecretURLParams
// to the URL value and returns the modified error. If not, the error is returned unchanged.
func MaskURLError(e error) error {
	var ue *url.Error
	ok := errors.As(e, &ue)
	if !ok {
		return e
	}
	ue.URL = MaskSecretURLParams(ue.URL)
	return ue
}
