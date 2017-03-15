package service

import (
	"context"
	"strconv"

	"github.com/kolide/kolide/server/kolide"
)

func (vm validationMiddleware) ImportConfig(ctx context.Context, cfg *kolide.ImportConfig) (*kolide.ImportConfigResponse, error) {
	var invalid invalidArgumentError
	vm.validateConfigOptions(cfg, &invalid)
	vm.validatePacks(cfg, &invalid)
	vm.validateDecorator(cfg, &invalid)
	vm.validateYARA(cfg, &invalid)
	if invalid.HasErrors() {
		return nil, invalid
	}
	return vm.Service.ImportConfig(ctx, cfg)
}

func (vm validationMiddleware) validateYARA(cfg *kolide.ImportConfig, argErrs *invalidArgumentError) {
	if cfg.YARA != nil {
		if cfg.YARA.FilePaths == nil {
			argErrs.Append("yara", "missing file_paths")
			return
		}
		if cfg.YARA.Signatures == nil {
			argErrs.Append("yara", "missing signatures")
		}
		for fileSection, sigs := range cfg.YARA.FilePaths {
			if cfg.FileIntegrityMonitoring == nil {
				argErrs.Append("yara", "missing file paths section")
				return
			}
			if _, ok := cfg.FileIntegrityMonitoring[fileSection]; !ok {
				argErrs.Appendf("yara", "missing referenced file_paths section '%s'", fileSection)
			}
			for _, sig := range sigs {
				if _, ok := cfg.YARA.Signatures[sig]; !ok {
					argErrs.Appendf(
						"yara",
						"missing signature '%s' referenced in '%s'",
						sig,
						fileSection,
					)
				}
			}
		}
	}
}

func (vm validationMiddleware) validateDecorator(cfg *kolide.ImportConfig, argErrs *invalidArgumentError) {
	if cfg.Decorators != nil {
		for str := range cfg.Decorators.Interval {
			val, err := strconv.ParseInt(str, 10, 32)
			if err != nil {
				argErrs.Appendf("decorators", "interval '%s' must be an integer", str)
				continue
			}
			if val%60 != 0 {

				argErrs.Appendf("decorators", "interval '%d' must be divisible by 60", val)
			}
		}
	}
}

func (vm validationMiddleware) validateConfigOptions(cfg *kolide.ImportConfig, argErrs *invalidArgumentError) {
	if cfg.Options != nil {
		for optName, optValue := range cfg.Options {
			opt, err := vm.ds.OptionByName(string(optName))
			if err != nil {
				// skip validation for an option we don't know about, this will generate
				// a warning in the service layer
				continue
			}
			if !opt.SameType(optValue) {
				argErrs.Appendf("options", "invalid type for '%s'", optName)
			}
		}
	}
}

func (vm validationMiddleware) validatePacks(cfg *kolide.ImportConfig, argErrs *invalidArgumentError) {
	if cfg.Packs != nil {
		for packName, pack := range cfg.Packs {
			// if glob packs is defined we expect at least one external pack
			if packName == kolide.GlobPacks {
				if len(cfg.GlobPackNames) == 0 {
					argErrs.Append("external_packs", "missing glob packs")
					continue
				}
				// make sure that each glob pack has JSON content
				for _, p := range cfg.GlobPackNames {
					if _, ok := cfg.ExternalPacks[p]; !ok {
						argErrs.Appendf("external_packs", "missing content for '%s'", p)
					}
				}
				continue
			}
			// if value is a string we expect a file path, in this case, the user has to supply the
			// contents of said file which we store in ExternalPacks, if it's not there we need to
			// raise an error
			switch pack.(type) {
			case string:
				if _, ok := cfg.ExternalPacks[packName]; !ok {
					argErrs.Appendf("external_packs", "missing content for '%s'", packName)
				}
			}
		}
	}
}
