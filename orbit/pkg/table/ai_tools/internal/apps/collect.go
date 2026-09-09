package apps

// This file holds the platform-neutral half of app collection: matching a
// discovered candidate against the known-app list, and deciding which of several
// discoveries of the same app wins.
//
// It lives outside the build-tagged files so those precedence rules can be
// tested on any platform; only the Windows build calls it today, where a single
// app can legitimately be discovered by two different scans of two different
// registry namespaces.

// appCandidate is one potential app a platform scan discovered, before it is
// matched against the known-app list and deduplicated.
type appCandidate struct {
	MatchTokens []string // identifying strings matched against knownApps
	DisplayName string
	Vendor      string
	Version     string
	Path        string
	Scope       string // system | user
	Source      string // PlatformSource
}

// appCollector accumulates candidates and keeps the first match for each known
// app, so discovery order is what encodes precedence:
//
//   - machine-wide registry roots are scanned before per-user ones, so an app
//     installed both ways is reported with scope "system";
//   - uninstall entries are scanned before MSIX packages, so an app registered
//     both ways keeps the richer DisplayVersion/InstallLocation metadata.
//
// Every scan on a platform shares one collector. Giving each scan its own would
// let the same app be reported twice under the same name, which the table has no
// way to reconcile.
type appCollector struct {
	seen map[string]struct{}
	out  []App
}

func newAppCollector() *appCollector {
	return &appCollector{seen: map[string]struct{}{}}
}

// wants reports whether tokens identify a known app that has not been collected
// yet. Callers use it to skip expensive metadata reads for candidates that would
// be discarded anyway.
func (c *appCollector) wants(tokens ...string) bool {
	k, ok := matchKnown(tokens...)
	if !ok {
		return false
	}
	_, dup := c.seen[k.name]
	return !dup
}

// add records cand unless it matches no known app, or that app was already
// collected by an earlier (higher-precedence) scan. It reports whether the
// candidate was added.
func (c *appCollector) add(cand appCandidate) bool {
	k, ok := matchKnown(cand.MatchTokens...)
	if !ok {
		return false
	}
	if _, dup := c.seen[k.name]; dup {
		return false
	}
	c.seen[k.name] = struct{}{}
	c.out = append(c.out, App{
		Name:           k.name,
		DisplayName:    cand.DisplayName,
		Vendor:         cand.Vendor,
		Version:        cand.Version,
		Path:           cand.Path,
		PlatformSource: cand.Source,
		Scope:          cand.Scope,
	})
	return true
}

// apps returns the collected apps in discovery order.
func (c *appCollector) apps() []App { return c.out }
