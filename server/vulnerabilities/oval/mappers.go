package oval

import (
	"fmt"
	"strconv"
	"strings"

	oval_input "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/input"
	oval_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/parsed"
)

// Discards the Namespace part of an OVAL id attr,
// returning only the last numeric portion
func extractId(idStr string) (int, error) {
	idParts := strings.Split(idStr, ":")
	return strconv.Atoi(idParts[len(idParts)-1])
}

func mapDefinition(i oval_input.DefinitionXML) (*oval_parsed.Definition, error) {
	r := oval_parsed.Definition{}

	for _, cve := range i.CVEs {
		r.Vulnerabilities = append(r.Vulnerabilities, cve.Id)
	}

	c, err := mapCriteria(i.Criteria)
	if err != nil {
		return nil, err
	}
	r.Criteria = c

	return &r, nil
}

func mapCriteria(i oval_input.CriteriaXML) (*oval_parsed.Criteria, error) {
	criteria := oval_parsed.Criteria{
		Operator: oval_parsed.NewOperatorType(i.Operator).Negate(i.Negate),
	}

	for _, criterion := range i.Criteriums {
		id, err := extractId(criterion.TestId)
		if err != nil {
			return nil, err
		}
		criteria.Criteriums = append(criteria.Criteriums, id)
	}

	for _, ic := range i.Criterias {
		mC, err := mapCriteria(ic)
		if err != nil {
			return nil, err
		}
		criteria.Criterias = append(criteria.Criterias, mC)
	}

	return &criteria, nil
}

func mapPackageTest(i oval_input.DpkgInfoTestXML) (int, *oval_parsed.DpkgInfoTest, error) {
	id, err := extractId(i.Id)
	if err != nil {
		return 0, nil, err
	}

	tst := oval_parsed.DpkgInfoTest{
		ObjectMatch:   oval_parsed.NewObjectMatchType(i.CheckExistence),
		StateMatch:    oval_parsed.NewStateMatchType(i.Check),
		StateOperator: oval_parsed.NewOperatorType(i.StateOperator),
	}

	return id, &tst, nil
}

func mapPackageState(sta oval_input.DpkgStateXML) ([]oval_parsed.ObjectStateEvrString, error) {
	var r []oval_parsed.ObjectStateEvrString

	if sta.Name != nil ||
		sta.Arch != nil ||
		sta.Epoch != nil ||
		sta.Version != nil {
		return nil, fmt.Errorf("only evr state definitions are supported")
	}

	if sta.Evr != nil {
		r = append(r, oval_parsed.NewObjectState(sta.Evr.Op, sta.Evr.Value))
	}

	return r, nil
}

func mapPackageObject(obj oval_input.DpkgObjectXML, vars map[string]oval_input.ConstantVariableXML) ([]string, error) {
	// Test objects can define their 'name' in one of two ways:
	// 1. Inline:
	// <:object ...>
	//      <:name>software name</:name>
	// </:object>
	//
	// 2. As a variable reference:
	// <:object ...>
	// 		<:name var_ref="var:200224390000000" var_check="at least one" />
	// </:object>

	// Check whether the name was defined inline
	if obj.Name.Value != "" {
		return []string{obj.Name.Value}, nil
	}

	var r []string
	// If not, the name should be defined as a variable
	variable, ok := vars[obj.Name.VarRef]
	if !ok {
		return nil, fmt.Errorf("variable not found %s", obj.Name.VarRef)
	}

	// Normally the variable for a test object contains a single value, but according to the specs,
	// it can contain multiple values.
	r = append(r, variable.Values...)

	return r, nil
}
