package oval

import (
	"fmt"
	"strconv"
	"strings"

	oval_input "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/input"
	oval_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/parsed"
)

// extractId discards the Namespace part of an OVAL id attr, returning only the last numeric portion.
func extractId(idStr string) (int, error) {
	idParts := strings.Split(idStr, ":")
	return strconv.Atoi(idParts[len(idParts)-1])
}

// mapDefinition maps a DefinitionXML into a Definition will error out if the definition contains
// no Vulnerabilities.
func mapDefinition(i oval_input.DefinitionXML) (*oval_parsed.Definition, error) {
	if len(i.Vulnerabilities) == 0 {
		return nil, fmt.Errorf("definition contains no vulnerabilities")
	}

	r := oval_parsed.Definition{}

	for _, vuln := range i.Vulnerabilities {
		r.Vulnerabilities = append(r.Vulnerabilities, vuln.Id)
	}

	c, err := mapCriteria(i.Criteria)
	if err != nil {
		return nil, err
	}
	r.Criteria = c

	return &r, nil
}

// mapCriteria maps a CriteriaXML into a Criteria, will error out if any Criterion is missing its id
// or if any of the Criteriums is empty.
func mapCriteria(i oval_input.CriteriaXML) (*oval_parsed.Criteria, error) {
	if len(i.Criteriums) == 0 {
		return nil, fmt.Errorf("invalid Criteria, Criteriums missing")
	}

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

// mapDpkgInfoTest maps a DpkgInfoTestXML returning the test id along side the mapped DpkgInfoTest
func mapDpkgInfoTest(i oval_input.DpkgInfoTestXML) (int, *oval_parsed.DpkgInfoTest, error) {
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

// mapDpkgInfoState maps a DpkgInfoStateXML into an EVR string. The state of an object defines
// the different information that can be used to evaluate the specified DPKG package. All Ubuntu
// OVAL definitions seem to only use Evr strings to define object state, that's why only Evr support
// was added at the moment. Adding support for `Name`, `Epoch` and `Version` should be trivial - in
// the case of `Arch`, it should be straightforward as well as long as the information we have in the
// `software` table is accurate. This will error out if object state is defined using anything else
// than an `Evr` string.
func mapDpkgInfoState(sta oval_input.DpkgInfoStateXML) (*oval_parsed.ObjectStateEvrString, error) {
	if sta.Name != nil ||
		sta.Arch != nil ||
		sta.Epoch != nil ||
		sta.Version != nil ||
		sta.Evr == nil {
		return nil, fmt.Errorf("only evr state definitions are supported")
	}

	r := oval_parsed.NewObjectState(sta.Evr.Op, sta.Evr.Value)
	return &r, nil
}

// mapDpkgInfoObject maps a DpkgInfoObjectXML into one or more object names.
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
func mapDpkgInfoObject(
	obj oval_input.DpkgInfoObjectXML,
	vars map[string]oval_input.ConstantVariableXML,
) ([]string, error) {
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

	// Normally the variable for a test object contains a single value but, according to the specs,
	// it can contain multiple values
	r = append(r, variable.Values...)

	return r, nil
}
