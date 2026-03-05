//go:build windows

package enforcement

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"os/exec"
	"strings"
)

type auditSetting struct {
	Subcategory  string `json:"subcategory" yaml:"subcategory"`
	IncludeValue string `json:"include" yaml:"include"`
	CISRef       string `json:"cis_ref,omitempty" yaml:"cis_ref,omitempty"`
}

type auditPolicy struct {
	AuditPolicy []auditSetting `json:"audit_policy" yaml:"audit_policy"`
}

// AuditHandler enforces audit policy settings via auditpol.exe.
type AuditHandler struct{}

func NewAuditHandler() *AuditHandler { return &AuditHandler{} }
func (h *AuditHandler) Name() string { return "audit" }

func (h *AuditHandler) Diff(ctx context.Context, rawPolicy []byte) ([]DiffResult, error) {
	var policy auditPolicy
	if err := yaml.Unmarshal(rawPolicy, &policy); err != nil {
		return nil, fmt.Errorf("parsing audit policy: %w", err)
	}

	if len(policy.AuditPolicy) == 0 {
		return nil, nil
	}

	current, err := getAuditPolicy(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting current audit policy: %w", err)
	}

	var results []DiffResult
	for _, s := range policy.AuditPolicy {
		currentValue := current[s.Subcategory]
		compliant := strings.EqualFold(currentValue, s.IncludeValue)
		results = append(results, DiffResult{
			SettingName:  s.Subcategory,
			Category:     "audit",
			CISRef:       s.CISRef,
			DesiredValue: s.IncludeValue,
			CurrentValue: currentValue,
			Compliant:    compliant,
		})
	}

	return results, nil
}

func (h *AuditHandler) Apply(ctx context.Context, rawPolicy []byte) ([]ApplyResult, error) {
	var policy auditPolicy
	if err := yaml.Unmarshal(rawPolicy, &policy); err != nil {
		return nil, fmt.Errorf("parsing audit policy: %w", err)
	}

	if len(policy.AuditPolicy) == 0 {
		return nil, nil
	}

	var results []ApplyResult
	for _, s := range policy.AuditPolicy {
		args := []string{"/set", "/subcategory:" + s.Subcategory}
		include := strings.ToLower(s.IncludeValue)
		switch include {
		case "success":
			args = append(args, "/success:enable", "/failure:disable")
		case "failure":
			args = append(args, "/success:disable", "/failure:enable")
		case "success and failure", "success, failure":
			args = append(args, "/success:enable", "/failure:enable")
		case "no auditing":
			args = append(args, "/success:disable", "/failure:disable")
		default:
			results = append(results, ApplyResult{
				SettingName: s.Subcategory,
				Category:    "audit",
				Success:     false,
				Error:       fmt.Sprintf("unsupported audit value: %s", s.IncludeValue),
			})
			continue
		}

		cmd := exec.CommandContext(ctx, "auditpol.exe", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			results = append(results, ApplyResult{
				SettingName: s.Subcategory,
				Category:    "audit",
				Success:     false,
				Error:       fmt.Sprintf("auditpol: %v: %s", err, string(output)),
			})
		} else {
			results = append(results, ApplyResult{
				SettingName: s.Subcategory,
				Category:    "audit",
				Success:     true,
			})
		}
	}

	return results, nil
}

func getAuditPolicy(ctx context.Context) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "auditpol.exe", "/get", "/category:*", "/r")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("auditpol get: %w", err)
	}

	result := make(map[string]string)
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		parts := strings.Split(line, ",")
		if len(parts) >= 4 {
			subcategory := strings.TrimSpace(parts[2])
			value := strings.TrimSpace(parts[3])
			if subcategory != "" {
				result[subcategory] = value
			}
		}
	}
	return result, nil
}
