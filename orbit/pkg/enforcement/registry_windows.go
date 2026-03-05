//go:build windows

package enforcement

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
	"gopkg.in/yaml.v3"
)

// RegistrySetting represents a single registry enforcement setting from the
// YAML policy.
type RegistrySetting struct {
	Path   string `json:"path" yaml:"path"`
	Name   string `json:"name" yaml:"name"`
	Type   string `json:"type" yaml:"type"`
	Value  any    `json:"value" yaml:"value"`
	Ensure string `json:"ensure,omitempty" yaml:"ensure,omitempty"`
	CISRef string `json:"cis_ref,omitempty" yaml:"cis_ref,omitempty"`
}

type registryPolicy struct {
	Registry []RegistrySetting `json:"registry" yaml:"registry"`
}

// RegistryHandler enforces registry-based security settings on Windows.
type RegistryHandler struct{}

func NewRegistryHandler() *RegistryHandler { return &RegistryHandler{} }
func (h *RegistryHandler) Name() string    { return "registry" }

func (h *RegistryHandler) Diff(ctx context.Context, rawPolicy []byte) ([]DiffResult, error) {
	var policy registryPolicy
	if err := yaml.Unmarshal(rawPolicy, &policy); err != nil {
		return nil, fmt.Errorf("parsing registry policy: %w", err)
	}

	if len(policy.Registry) == 0 {
		return nil, nil
	}

	var results []DiffResult
	for _, s := range policy.Registry {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		rootKey, subPath, err := parseRegistryPath(s.Path)
		if err != nil {
			results = append(results, DiffResult{
				SettingName:  s.Name,
				Category:     "registry",
				CISRef:       s.CISRef,
				DesiredValue: fmt.Sprintf("%v", s.Value),
				CurrentValue: "error: " + err.Error(),
				Compliant:    false,
			})
			continue
		}

		hkey, err := rootKeyToHKEY(rootKey)
		if err != nil {
			results = append(results, DiffResult{
				SettingName:  s.Name,
				Category:     "registry",
				CISRef:       s.CISRef,
				DesiredValue: fmt.Sprintf("%v", s.Value),
				CurrentValue: "error: " + err.Error(),
				Compliant:    false,
			})
			continue
		}

		k, err := registry.OpenKey(hkey, subPath, registry.QUERY_VALUE)
		if err != nil {
			ensure := s.Ensure
			if ensure == "" {
				ensure = "present"
			}
			compliant := ensure == "absent"
			results = append(results, DiffResult{
				SettingName:  s.Name,
				Category:     "registry",
				CISRef:       s.CISRef,
				DesiredValue: fmt.Sprintf("%v", s.Value),
				CurrentValue: "(not found)",
				Compliant:    compliant,
			})
			continue
		}

		currentValue, err := readRegistryValue(k, s.Name, s.Type)
		k.Close()

		if err != nil {
			results = append(results, DiffResult{
				SettingName:  s.Name,
				Category:     "registry",
				CISRef:       s.CISRef,
				DesiredValue: fmt.Sprintf("%v", s.Value),
				CurrentValue: "error: " + err.Error(),
				Compliant:    false,
			})
			continue
		}

		desiredStr := fmt.Sprintf("%v", s.Value)
		currentStr := fmt.Sprintf("%v", currentValue)
		compliant := desiredStr == currentStr

		results = append(results, DiffResult{
			SettingName:  s.Name,
			Category:     "registry",
			CISRef:       s.CISRef,
			DesiredValue: desiredStr,
			CurrentValue: currentStr,
			Compliant:    compliant,
		})
	}

	return results, nil
}

func (h *RegistryHandler) Apply(ctx context.Context, rawPolicy []byte) ([]ApplyResult, error) {
	var policy registryPolicy
	if err := yaml.Unmarshal(rawPolicy, &policy); err != nil {
		return nil, fmt.Errorf("parsing registry policy: %w", err)
	}

	if len(policy.Registry) == 0 {
		return nil, nil
	}

	var results []ApplyResult
	for _, s := range policy.Registry {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		rootKey, subPath, err := parseRegistryPath(s.Path)
		if err != nil {
			results = append(results, ApplyResult{
				SettingName: s.Name,
				Category:    "registry",
				Success:     false,
				Error:       err.Error(),
			})
			continue
		}

		hkey, err := rootKeyToHKEY(rootKey)
		if err != nil {
			results = append(results, ApplyResult{
				SettingName: s.Name,
				Category:    "registry",
				Success:     false,
				Error:       err.Error(),
			})
			continue
		}

		ensure := s.Ensure
		if ensure == "" {
			ensure = "present"
		}

		if ensure == "absent" {
			k, err := registry.OpenKey(hkey, subPath, registry.SET_VALUE)
			if err == nil {
				err = k.DeleteValue(s.Name)
				k.Close()
			}
			if err != nil {
				results = append(results, ApplyResult{
					SettingName: s.Name,
					Category:    "registry",
					Success:     false,
					Error:       err.Error(),
				})
			} else {
				results = append(results, ApplyResult{
					SettingName: s.Name,
					Category:    "registry",
					Success:     true,
				})
			}
			continue
		}

		// ensure == "present" - create or update
		k, _, err := registry.CreateKey(hkey, subPath, registry.SET_VALUE)
		if err != nil {
			results = append(results, ApplyResult{
				SettingName: s.Name,
				Category:    "registry",
				Success:     false,
				Error:       fmt.Sprintf("create key: %v", err),
			})
			continue
		}

		err = writeRegistryValue(k, s.Name, s.Type, s.Value)
		k.Close()

		if err != nil {
			results = append(results, ApplyResult{
				SettingName: s.Name,
				Category:    "registry",
				Success:     false,
				Error:       err.Error(),
			})
		} else {
			results = append(results, ApplyResult{
				SettingName: s.Name,
				Category:    "registry",
				Success:     true,
			})
		}
	}

	return results, nil
}

func parseRegistryPath(fullPath string) (rootKey, subPath string, err error) {
	idx := strings.Index(fullPath, `\`)
	if idx < 0 {
		return "", "", fmt.Errorf("invalid registry path %q: missing backslash", fullPath)
	}
	return fullPath[:idx], fullPath[idx+1:], nil
}

func rootKeyToHKEY(rootKey string) (registry.Key, error) {
	switch strings.ToUpper(rootKey) {
	case "HKLM", "HKEY_LOCAL_MACHINE":
		return registry.LOCAL_MACHINE, nil
	case "HKCU", "HKEY_CURRENT_USER":
		return registry.CURRENT_USER, nil
	case "HKCR", "HKEY_CLASSES_ROOT":
		return registry.CLASSES_ROOT, nil
	case "HKU", "HKEY_USERS":
		return registry.USERS, nil
	default:
		return 0, fmt.Errorf("unsupported root key: %s", rootKey)
	}
}

func readRegistryValue(k registry.Key, name, regType string) (any, error) {
	switch strings.ToLower(regType) {
	case "dword", "reg_dword":
		val, _, err := k.GetIntegerValue(name)
		return val, err
	case "string", "sz", "reg_sz":
		val, _, err := k.GetStringValue(name)
		return val, err
	case "multi_string", "multi_sz", "reg_multi_sz":
		val, _, err := k.GetStringsValue(name)
		return strings.Join(val, ","), err
	default:
		return nil, fmt.Errorf("unsupported registry type: %s", regType)
	}
}

func writeRegistryValue(k registry.Key, name, regType string, value any) error {
	switch strings.ToLower(regType) {
	case "dword", "reg_dword":
		var v uint32
		switch val := value.(type) {
		case float64:
			v = uint32(val)
		case int:
			v = uint32(val)
		case int64:
			v = uint32(val)
		default:
			return fmt.Errorf("cannot convert %T to DWORD", value)
		}
		return k.SetDWordValue(name, v)
	case "string", "sz", "reg_sz":
		s, ok := value.(string)
		if !ok {
			s = fmt.Sprintf("%v", value)
		}
		return k.SetStringValue(name, s)
	default:
		return fmt.Errorf("unsupported registry type for write: %s", regType)
	}
}
