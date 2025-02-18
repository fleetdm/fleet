package customcve

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

var (
	MissingCVEsErr              = errors.New("CVEs must be specified")
	MissingNameLikeMatch        = errors.New("NameLikeMatch must be specified")
	MissingResolvedInVersionErr = errors.New("ResolvedInVersion must be specified")
)

// CVEMatchingRuleSpec contains custom matching rules for matching software
// with a list of CVEs.  These rules address false negatives in the NVD data.
// Add an interface if you want to add more rule types.
type CVEMatchingRule struct {
	NameLikeMatch     string   // Name of software to match (like match)
	SourceMatch       string   // Source of software to match (exact match)
	CVEs              []string // List of CVEs to assign to software
	ResolvedInVersion string   // Version of software that resolves the CVEs
}

type CVEMatchingRules []CVEMatchingRule

// getCVEMatchingRules returns a list of custom rules for matching software with CVEs
// Currently only supporting CVEMatchingRules, but can be extended to support other types.
// Append new rules here.
func getCVEMatchingRules() CVEMatchingRules {
	return []CVEMatchingRule{
		// June 11 2024 Office 365 Vulnerabilities
		// https://learn.microsoft.com/en-us/officeupdates/microsoft365-apps-security-updates
		{
			NameLikeMatch:     "Microsoft 365",
			SourceMatch:       "programs",
			CVEs:              []string{"CVE-2024-30101", "CVE-2024-30102", "CVE-2024-30103", "CVE-2024-30104"},
			ResolvedInVersion: "16.0.17628.20144",
		},
		// July 9 2024 Office 365 Vulnerabilities
		// https://learn.microsoft.com/en-us/officeupdates/microsoft365-apps-security-updates
		{
			NameLikeMatch:     "Microsoft 365",
			SourceMatch:       "programs",
			CVEs:              []string{"CVE-2023-38545", "CVE-2024-38020", "CVE-2024-38021"},
			ResolvedInVersion: "16.0.17726.20160",
		},
		// August 13 2024 Office 365 Vulnerabilities
		// https://learn.microsoft.com/en-us/officeupdates/microsoft365-apps-security-updates
		{
			NameLikeMatch: "Microsoft 365",
			SourceMatch:   "programs",
			CVEs: []string{
				"CVE-2024-38172",
				"CVE-2024-38170",
				"CVE-2024-38173",
				"CVE-2024-38171",
				"CVE-2024-38189",
				"CVE-2024-38169",
				"CVE-2024-38200",
			},
			ResolvedInVersion: "16.0.17830.20166",
		},
	}
}

func (r CVEMatchingRule) match(ctx context.Context, ds fleet.Datastore) ([]fleet.SoftwareVulnerability, error) {
	var vulns []fleet.SoftwareVulnerability
	filter := fleet.VulnSoftwareFilter{
		Name:   r.NameLikeMatch,
		Source: r.SourceMatch,
	}
	software, err := ds.ListSoftwareForVulnDetection(ctx, filter)
	if err != nil {
		return nil, err
	}

	for _, s := range software {
		if nvd.SmartVerCmp(s.Version, r.ResolvedInVersion) < 0 {
			for _, cve := range r.CVEs {
				vulns = append(vulns, fleet.SoftwareVulnerability{
					SoftwareID:        s.ID,
					CVE:               cve,
					ResolvedInVersion: &r.ResolvedInVersion,
				})
			}
		}
	}

	return vulns, nil
}

func (r CVEMatchingRule) validate() error {
	if len(r.CVEs) == 0 {
		return MissingCVEsErr
	}

	if r.NameLikeMatch == "" {
		return MissingNameLikeMatch
	}

	if r.ResolvedInVersion == "" {
		return MissingResolvedInVersionErr
	}

	return nil
}

// ValidateAll returns an error if any rule in the list fails to validate
func (r CVEMatchingRules) ValidateAll() error {
	for i, rule := range r {
		if err := rule.validate(); err != nil {
			return fmt.Errorf("invalid rule %d: %v", i, err)
		}
	}
	return nil
}

// CheckCustomVulnerabilities matches software against custom rules and inserts vulnerabilities
func CheckCustomVulnerabilities(ctx context.Context, ds fleet.Datastore, logger log.Logger, periodicity time.Duration) ([]fleet.SoftwareVulnerability, error) {
	rules := getCVEMatchingRules()
	if err := rules.ValidateAll(); err != nil {
		return nil, fmt.Errorf("invalid rules: %w", err)
	}

	var vulns []fleet.SoftwareVulnerability
	for i, rule := range rules {
		v, err := rule.match(ctx, ds)
		if err != nil {
			level.Error(logger).Log("msg", "Error matching rule", "ruleIndex", i, "err", err)
			continue
		}
		vulns = append(vulns, v...)
	}

	var newVulns []fleet.SoftwareVulnerability
	for _, v := range vulns {
		ok, err := ds.InsertSoftwareVulnerability(ctx, v, fleet.CustomSource)
		if err != nil {
			level.Error(logger).Log("msg", "Error inserting software vulnerability", "err", err)
			continue
		}
		if ok {
			newVulns = append(newVulns, v)
		}
	}

	if err := ds.DeleteOutOfDateVulnerabilities(ctx, fleet.CustomSource, 2*periodicity); err != nil {
		level.Error(logger).Log("msg", "Error deleting out of date vulnerabilities", "err", err)
	}

	return newVulns, nil
}
