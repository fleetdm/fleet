package service

import (
	"github.com/ghodss/yaml"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (svc service) ApplyOptionsYaml(yml string) error {
	var conf kolide.OptionsYaml
	err := yaml.Unmarshal([]byte(yml), &conf)
	if err != nil {
		return errors.Wrap(err, "unmarshal options YAML")
	}

	if conf.Kind != kolide.OptionsSpecKind {
		return errors.Errorf("expected kind '%s', got '%s'", kolide.OptionsSpecKind, conf.Kind)
	}

	err = svc.ds.ApplyOptions(&conf.Spec)
	return errors.Wrap(err, "apply options")
}

func (svc service) GetOptionsYaml() (string, error) {
	spec, err := svc.ds.GetOptions()
	if err != nil {
		return "", errors.Wrap(err, "get options from datastore")
	}

	ymlObj := kolide.OptionsYaml{
		ApiVersion: kolide.ApiVersion,
		Kind:       kolide.OptionsSpecKind,
		Spec:       *spec,
	}

	yml, err := yaml.Marshal(ymlObj)
	if err != nil {
		return "", errors.Wrap(err, "marshal options yaml")
	}
	return string(yml), nil
}
