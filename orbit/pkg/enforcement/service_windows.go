//go:build windows

package enforcement

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"strings"

	"golang.org/x/sys/windows/svc/mgr"
)

type serviceSetting struct {
	Name        string `json:"name" yaml:"name"`
	StartupType string `json:"startup_type" yaml:"startup_type"`
	CISRef      string `json:"cis_ref,omitempty" yaml:"cis_ref,omitempty"`
}

type servicePolicy struct {
	Services []serviceSetting `json:"services" yaml:"services"`
}

// ServiceHandler enforces Windows service startup type configurations via SCM.
type ServiceHandler struct{}

func NewServiceHandler() *ServiceHandler { return &ServiceHandler{} }
func (h *ServiceHandler) Name() string   { return "service" }

func (h *ServiceHandler) Diff(ctx context.Context, rawPolicy []byte) ([]DiffResult, error) {
	var policy servicePolicy
	if err := yaml.Unmarshal(rawPolicy, &policy); err != nil {
		return nil, fmt.Errorf("parsing service policy: %w", err)
	}

	if len(policy.Services) == 0 {
		return nil, nil
	}

	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("connecting to service manager: %w", err)
	}
	defer m.Disconnect()

	var results []DiffResult
	for _, s := range policy.Services {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		svc, err := m.OpenService(s.Name)
		if err != nil {
			results = append(results, DiffResult{
				SettingName:  s.Name,
				Category:     "service",
				CISRef:       s.CISRef,
				DesiredValue: s.StartupType,
				CurrentValue: "(not found)",
				Compliant:    false,
			})
			continue
		}

		cfg, err := svc.Config()
		svc.Close()

		if err != nil {
			results = append(results, DiffResult{
				SettingName:  s.Name,
				Category:     "service",
				CISRef:       s.CISRef,
				DesiredValue: s.StartupType,
				CurrentValue: "error: " + err.Error(),
				Compliant:    false,
			})
			continue
		}

		currentType := startTypeToString(cfg.StartType)
		compliant := strings.EqualFold(currentType, s.StartupType)

		results = append(results, DiffResult{
			SettingName:  s.Name,
			Category:     "service",
			CISRef:       s.CISRef,
			DesiredValue: s.StartupType,
			CurrentValue: currentType,
			Compliant:    compliant,
		})
	}

	return results, nil
}

func (h *ServiceHandler) Apply(ctx context.Context, rawPolicy []byte) ([]ApplyResult, error) {
	var policy servicePolicy
	if err := yaml.Unmarshal(rawPolicy, &policy); err != nil {
		return nil, fmt.Errorf("parsing service policy: %w", err)
	}

	if len(policy.Services) == 0 {
		return nil, nil
	}

	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("connecting to service manager: %w", err)
	}
	defer m.Disconnect()

	var results []ApplyResult
	for _, s := range policy.Services {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		svc, err := m.OpenService(s.Name)
		if err != nil {
			results = append(results, ApplyResult{
				SettingName: s.Name,
				Category:    "service",
				Success:     false,
				Error:       fmt.Sprintf("open service: %v", err),
			})
			continue
		}

		cfg, err := svc.Config()
		if err != nil {
			svc.Close()
			results = append(results, ApplyResult{
				SettingName: s.Name,
				Category:    "service",
				Success:     false,
				Error:       fmt.Sprintf("get config: %v", err),
			})
			continue
		}

		desired, err := stringToStartType(s.StartupType)
		if err != nil {
			svc.Close()
			results = append(results, ApplyResult{
				SettingName: s.Name,
				Category:    "service",
				Success:     false,
				Error:       err.Error(),
			})
			continue
		}

		cfg.StartType = desired
		err = svc.UpdateConfig(cfg)
		svc.Close()

		if err != nil {
			results = append(results, ApplyResult{
				SettingName: s.Name,
				Category:    "service",
				Success:     false,
				Error:       fmt.Sprintf("update config: %v", err),
			})
		} else {
			results = append(results, ApplyResult{
				SettingName: s.Name,
				Category:    "service",
				Success:     true,
			})
		}
	}

	return results, nil
}

func startTypeToString(startType uint32) string {
	switch startType {
	case mgr.StartAutomatic:
		return "automatic"
	case mgr.StartManual:
		return "manual"
	case mgr.StartDisabled:
		return "disabled"
	default:
		return fmt.Sprintf("unknown(%d)", startType)
	}
}

func stringToStartType(s string) (uint32, error) {
	switch strings.ToLower(s) {
	case "automatic", "auto":
		return mgr.StartAutomatic, nil
	case "manual":
		return mgr.StartManual, nil
	case "disabled":
		return mgr.StartDisabled, nil
	default:
		return 0, fmt.Errorf("unsupported startup type: %s", s)
	}
}
