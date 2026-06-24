package services

import (
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
	"github.com/fleetdm/fleet/tools/hangar/internal/perf"
	"github.com/fleetdm/fleet/tools/hangar/internal/perfconfig"
)

// PerfService exposes the osquery-perf template catalog. Mirrors perf.rs.
type PerfService struct{}

func (s *PerfService) PerfListTemplates() []perf.Template { return perf.Templates() }

// PerfConfigService exposes saved osquery-perf run configs. Mirrors perf_configs.rs.
type PerfConfigService struct{}

func (s *PerfConfigService) PerfConfigsList() ([]perfconfig.Config, error) {
	dir, err := paths.ConfigDir()
	if err != nil {
		return nil, err
	}
	return perfconfig.List(dir)
}
func (s *PerfConfigService) PerfConfigSave(config perfconfig.Config) (perfconfig.Config, error) {
	dir, err := paths.ConfigDir()
	if err != nil {
		return perfconfig.Config{}, err
	}
	return perfconfig.Save(dir, config, uint64(time.Now().UnixMilli()))
}
func (s *PerfConfigService) PerfConfigDelete(id string) error {
	dir, err := paths.ConfigDir()
	if err != nil {
		return err
	}
	return perfconfig.Delete(dir, id)
}
