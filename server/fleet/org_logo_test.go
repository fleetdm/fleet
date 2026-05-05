package fleet

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOrgLogoBytesSVG(t *testing.T) {
	t.Parallel()

	const minSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"></svg>`

	t.Run("accepts a minimal SVG", func(t *testing.T) {
		require.NoError(t, ValidateOrgLogoBytes([]byte(minSVG)))
	})

	t.Run("accepts an SVG with XML declaration and inline style", func(t *testing.T) {
		body := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">
  <style>.a { fill: red; }</style>
  <rect class="a" width="10" height="10"/>
</svg>`
		require.NoError(t, ValidateOrgLogoBytes([]byte(body)))
	})

	t.Run("accepts xlink:href fragment references", func(t *testing.T) {
		body := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><defs><circle id="c" r="5"/></defs><use xlink:href="#c"/></svg>`
		require.NoError(t, ValidateOrgLogoBytes([]byte(body)))
	})

	t.Run("accepts a real-world SVG (CSS logo from the public web)", func(t *testing.T) {
		body, err := os.ReadFile("testdata/icons/org_logo_css.svg")
		require.NoError(t, err)
		require.NoError(t, ValidateOrgLogoBytes(body))
	})

	t.Run("rejects oversized SVG before parsing", func(t *testing.T) {
		// Bytes don't need to be a real SVG — the size gate fires
		// first regardless of looksLikeSVG.
		body := append([]byte("<svg>"), bytes.Repeat([]byte("a"), int(OrgLogoMaxFileSize))...)
		err := ValidateOrgLogoBytes(body)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "100KB or less")
	})

	t.Run("rejects <script>", func(t *testing.T) {
		body := `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`
		err := ValidateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "<script>")
	})

	t.Run("rejects <foreignObject>", func(t *testing.T) {
		body := `<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><div xmlns="http://www.w3.org/1999/xhtml">x</div></foreignObject></svg>`
		err := ValidateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "foreignobject")
	})

	t.Run("rejects on* event handlers", func(t *testing.T) {
		body := `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><rect width="1" height="1"/></svg>`
		err := ValidateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event-handler")
	})

	t.Run("href/src URL schemes", func(t *testing.T) {
		// Allowlist: only fragment, relative, or http(s) are safe.
		// Blocklist would have to chase javascript: + vbscript: +
		// livescript: + mocha: + data: + file: + every future scheme,
		// which is what CodeQL's "incomplete URL scheme check"
		// warning is about.
		cases := []struct {
			name string
			href string
			ok   bool
		}{
			{"fragment", "#defs-id", true},
			{"relative", "./icon.png", true},
			{"https", "https://example.com/x.png", true},
			{"http", "http://example.com/x.png", true},
			{"javascript", "javascript:alert(1)", false},
			{"vbscript", "vbscript:msgbox(1)", false},
			{"data", "data:text/html,&lt;script&gt;a()&lt;/script&gt;", false},
			{"file", "file:///etc/passwd", false},
			{"livescript", "livescript:alert(1)", false},
			{"uppercase javascript", "JAVASCRIPT:alert(1)", false},
			{"leading whitespace + javascript", "  javascript:alert(1)", false},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				body := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><a xlink:href=%q><rect width="1" height="1"/></a></svg>`, c.href)
				err := ValidateOrgLogoBytes([]byte(body))
				if c.ok {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
					assert.Contains(t, err.Error(), "fragment")
				}
			})
		}
	})

	t.Run("rejects DOCTYPE", func(t *testing.T) {
		body := `<?xml version="1.0"?>
<!DOCTYPE svg [<!ENTITY xxe SYSTEM "file:///etc/passwd">]>
<svg xmlns="http://www.w3.org/2000/svg"><text>&xxe;</text></svg>`
		err := ValidateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "doctype")
	})

	t.Run("rejects malformed XML", func(t *testing.T) {
		body := `<svg xmlns="http://www.w3.org/2000/svg"><rect`
		err := ValidateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "valid SVG")
	})

	t.Run("rejects non-svg root that still tripped the sniffer", func(t *testing.T) {
		// looksLikeSVG just searches for "<svg" anywhere in the head;
		// a wrapper element must not let a document slip past with
		// the sniffer satisfied but the parsed root non-svg.
		body := `<html><body><svg/></body></html>`
		err := ValidateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "root")
	})
}

func TestContentTypeForOrgLogoSVG(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
		want string
	}{
		{"plain svg", `<svg xmlns="http://www.w3.org/2000/svg"/>`, "image/svg+xml"},
		{"svg after xml decl", `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"/>`, "image/svg+xml"},
		{"svg after BOM and whitespace", "\xEF\xBB\xBF\n  <svg/>", "image/svg+xml"},
		{"uppercase root tag", "<SVG/>", "image/svg+xml"},
		{"not an svg", `<html><body/></html>`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ContentTypeForOrgLogo([]byte(tc.body)))
		})
	}
}
