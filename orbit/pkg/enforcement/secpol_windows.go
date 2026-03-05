//go:build windows

package enforcement

import (
	"context"
	"gopkg.in/yaml.v3"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type secpolSetting struct {
	Category string `json:"category" yaml:"category"`
	Name     string `json:"name" yaml:"name"`
	Value    string `json:"value" yaml:"value"`
	CISRef   string `json:"cis_ref,omitempty" yaml:"cis_ref,omitempty"`
}

type secpolPolicy struct {
	LocalSecurityPolicy []secpolSetting `json:"local_security_policy" yaml:"local_security_policy"`
}

// SecpolHandler enforces local security policy settings via secedit.
type SecpolHandler struct{}

func NewSecpolHandler() *SecpolHandler { return &SecpolHandler{} }
func (h *SecpolHandler) Name() string  { return "secpol" }

func (h *SecpolHandler) Diff(ctx context.Context, rawPolicy []byte) ([]DiffResult, error) {
	var policy secpolPolicy
	if err := yaml.Unmarshal(rawPolicy, &policy); err != nil {
		return nil, fmt.Errorf("parsing secpol policy: %w", err)
	}

	if len(policy.LocalSecurityPolicy) == 0 {
		return nil, nil
	}

	// Export current security policy
	current, err := exportSecurityPolicy(ctx)
	if err != nil {
		return nil, fmt.Errorf("exporting current policy: %w", err)
	}

	var results []DiffResult
	for _, s := range policy.LocalSecurityPolicy {
		currentValue := current[s.Name]
		compliant := strings.EqualFold(currentValue, s.Value)
		results = append(results, DiffResult{
			SettingName:  s.Name,
			Category:     s.Category,
			CISRef:       s.CISRef,
			DesiredValue: s.Value,
			CurrentValue: currentValue,
			Compliant:    compliant,
		})
	}

	return results, nil
}

func (h *SecpolHandler) Apply(ctx context.Context, rawPolicy []byte) ([]ApplyResult, error) {
	var policy secpolPolicy
	if err := yaml.Unmarshal(rawPolicy, &policy); err != nil {
		return nil, fmt.Errorf("parsing secpol policy: %w", err)
	}

	if len(policy.LocalSecurityPolicy) == 0 {
		return nil, nil
	}

	// Build an INF template with the desired settings
	tmpDir, err := os.MkdirTemp("", "enforcement-secpol-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	infPath := filepath.Join(tmpDir, "apply.inf")
	dbPath := filepath.Join(tmpDir, "apply.sdb")

	// Group settings by INF section
	sections := make(map[string][]string)
	for _, s := range policy.LocalSecurityPolicy {
		section := categoryToSection(s.Category)
		sections[section] = append(sections[section], fmt.Sprintf("%s = %s", s.Name, s.Value))
	}

	var sb strings.Builder
	sb.WriteString("[Unicode]\nUnicode=yes\n[Version]\nsignature=\"$CHICAGO$\"\nRevision=1\n")
	for section, lines := range sections {
		sb.WriteString(fmt.Sprintf("[%s]\n", section))
		for _, line := range lines {
			sb.WriteString(line + "\n")
		}
	}

	if err := os.WriteFile(infPath, []byte(sb.String()), 0o600); err != nil {
		return nil, fmt.Errorf("writing INF file: %w", err)
	}

	// Apply via secedit
	cmd := exec.CommandContext(ctx, "secedit.exe", "/configure", "/db", dbPath, "/cfg", infPath, "/areas", "SECURITYPOLICY")
	output, err := cmd.CombinedOutput()

	var results []ApplyResult
	if err != nil {
		for _, s := range policy.LocalSecurityPolicy {
			results = append(results, ApplyResult{
				SettingName: s.Name,
				Category:    s.Category,
				Success:     false,
				Error:       fmt.Sprintf("secedit: %v: %s", err, string(output)),
			})
		}
	} else {
		for _, s := range policy.LocalSecurityPolicy {
			results = append(results, ApplyResult{
				SettingName: s.Name,
				Category:    s.Category,
				Success:     true,
			})
		}
	}

	return results, nil
}

func exportSecurityPolicy(ctx context.Context) (map[string]string, error) {
	tmpDir, err := os.MkdirTemp("", "enforcement-secpol-export-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	exportPath := filepath.Join(tmpDir, "export.inf")
	cmd := exec.CommandContext(ctx, "secedit.exe", "/export", "/cfg", exportPath)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("secedit export: %w", err)
	}

	data, err := os.ReadFile(exportPath)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			result[key] = val
		}
	}
	return result, nil
}

func categoryToSection(category string) string {
	switch category {
	case "Password Policy", "Account Lockout Policy", "Security Options":
		return "System Access"
	case "Event Audit":
		return "Event Audit"
	default:
		return "System Access"
	}
}
