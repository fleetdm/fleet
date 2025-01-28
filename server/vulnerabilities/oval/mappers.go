package oval

import (
	"errors"
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
		return nil, errors.New("definition contains no vulnerabilities")
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
// or if any of the Criteria contains no Criteriums nor nested Criterias
func mapCriteria(i oval_input.CriteriaXML) (*oval_parsed.Criteria, error) {
	if len(i.Criteriums) == 0 && len(i.Criterias) == 0 {
		return nil, errors.New("invalid Criteria, no Criteriums nor nested Criterias found")
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

// mapPackageInfoTestObject maps a PackageInfoTestObjectXML into one or more object names.
// Test objects can define their 'name' in one of two ways:
// 1. Inline:
// <:object ...>
//
//	<:name>software name</:name>
//
// </:object>
//
// 2. As a variable reference:
// <:object ...>
//
//	<:name var_ref="var:200224390000000" var_check="at least one" />
//
// </:object>
func mapPackageInfoTestObject(
	obj oval_input.PackageInfoTestObjectXML,
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

	// If the name is defined using a variable, it can contain multiple values
	r = append(r, variable.Values...)

	return r, nil
}

// -----------------
// RHEL
// -----------------

// mapRpmVerifyFileTest maps a RpmVerifyFileTestXML returning the test id along side the mapped RpmVerifyFileTest,
// will error out if the test id can not be parsed.
func mapRpmVerifyFileTest(i oval_input.RpmVerifyFileTestXML) (int, *oval_parsed.RpmVerifyFileTest, error) {
	id, err := extractId(i.Id)
	if err != nil {
		return 0, nil, err
	}

	tst := oval_parsed.RpmVerifyFileTest{
		ObjectMatch:   oval_parsed.NewObjectMatchType(i.CheckExistence),
		StateMatch:    oval_parsed.NewStateMatchType(i.Check),
		StateOperator: oval_parsed.NewOperatorType(i.StateOperator),
	}

	return id, &tst, nil
}

// mapRpmInfoTest maps a RpmInfoTestXML returning the test id along side the mapped RpmInfoTest,
// will error out if the test id can not be parsed.
func mapRpmInfoTest(i oval_input.RpmInfoTestXML) (int, *oval_parsed.RpmInfoTest, error) {
	id, err := extractId(i.Id)
	if err != nil {
		return 0, nil, err
	}

	tst := oval_parsed.RpmInfoTest{
		ObjectMatch:   oval_parsed.NewObjectMatchType(i.CheckExistence),
		StateMatch:    oval_parsed.NewStateMatchType(i.Check),
		StateOperator: oval_parsed.NewOperatorType(i.StateOperator),
	}

	return id, &tst, nil
}

// mapRpmVerifyFileObject maps a RpmVerifyFileObjectXML into file path (string), will error out if
// the `<filepath>` children element is not set or if any of the non supported element is set.
func mapRpmVerifyFileObject(i oval_input.RpmVerifyFileObjectXML) (*string, error) {
	if i.FilePath.Value == "" {
		return nil, errors.New("missing file path")
	}

	// The following properties are not used (since we are making an assertion against the contents
	// of a file), but they are required according to the specs - they should be present but empty
	if i.Name.Value != "" ||
		i.Epoch.Value != "" ||
		i.Version.Value != "" ||
		i.Release.Value != "" ||
		i.Arch.Value != "" {
		return nil, errors.New("invalid RPM verify file object specified")
	}

	filepath := i.FilePath.Value
	return &filepath, nil
}

// mapRpmVerifyFileState maps a RpmVerifyFileStateXML to an ObjectInfoState, will error out if a non
// supported attribute is found
func mapRpmVerifyFileState(sta oval_input.RpmVerifyFileStateXML) (*oval_parsed.ObjectInfoState, error) {
	if sta.SizeDiffers != nil ||
		sta.ModeDiffers != nil ||
		sta.Md5Differs != nil ||
		sta.DeviceDiffers != nil ||
		sta.LinkMismatch != nil ||
		sta.OwnershipDiffers != nil ||
		sta.GroupDiffers != nil ||
		sta.MtimeDiffers != nil ||
		sta.CapabilitiesDiffer != nil ||
		sta.ConfigurationFile != nil ||
		sta.GhostFile != nil ||
		sta.LicenseFile != nil ||
		sta.ReadmeFile != nil ||
		sta.Arch != nil ||
		sta.Epoch != nil ||
		sta.ExtendedName != nil {
		return nil, errors.New("invalid RPM verify file state specified")
	}
	r := oval_parsed.ObjectInfoState{}

	if sta.Name != nil {
		name := oval_parsed.NewObjectStateString(sta.Name.Op, sta.Name.Value)
		r.Name = &name
	}
	if sta.Version != nil {
		ver := oval_parsed.NewObjectStateSimpleValue(sta.Version.Datatype, sta.Version.Op, sta.Version.Value)
		r.Version = &ver
	}

	if sta.Operator != nil {
		r.Operator = oval_parsed.NewOperatorType(*sta.Operator)
	} else {
		r.Operator = oval_parsed.And
	}

	return &r, nil
}

// mapRpmInfoState maps a RpmInfoStateXML into an ObjectInfoState, will error out if one of the
// non-supported object states is specified
func mapRpmInfoState(sta oval_input.RpmInfoStateXML) (*oval_parsed.ObjectInfoState, error) {
	if sta.Filepath != nil {
		return nil, errors.New("object state based on filepath not supported")
	}

	r := oval_parsed.ObjectInfoState{}

	if sta.Name != nil {
		name := oval_parsed.NewObjectStateString(sta.Name.Op, sta.Name.Value)
		r.Name = &name
	}
	if sta.Arch != nil {
		arch := oval_parsed.NewObjectStateString(sta.Arch.Op, sta.Arch.Value)
		r.Arch = &arch
	}
	if sta.Epoch != nil {
		epoch := oval_parsed.NewObjectStateSimpleValue(sta.Epoch.Datatype, sta.Epoch.Op, sta.Epoch.Value)
		r.Epoch = &epoch
	}
	if sta.Release != nil {
		epoch := oval_parsed.NewObjectStateSimpleValue(sta.Release.Datatype, sta.Release.Op, sta.Release.Value)
		r.Release = &epoch
	}
	if sta.Version != nil {
		ver := oval_parsed.NewObjectStateSimpleValue(sta.Version.Datatype, sta.Version.Op, sta.Version.Value)
		r.Version = &ver
	}
	if sta.Evr != nil {
		evr := oval_parsed.NewObjectStateEvrString(sta.Evr.Op, sta.Evr.Value)
		r.Evr = &evr
	}
	if sta.SignatureKeyId != nil {
		sig := oval_parsed.NewObjectStateString(sta.SignatureKeyId.Op, sta.SignatureKeyId.Value)
		r.SignatureKeyId = &sig
	}
	if sta.ExtendedName != nil {
		extd := oval_parsed.NewObjectStateString(sta.ExtendedName.Op, sta.ExtendedName.Value)
		r.ExtendedName = &extd
	}

	if sta.Operator != nil {
		r.Operator = oval_parsed.NewOperatorType(*sta.Operator)
	} else {
		r.Operator = oval_parsed.And
	}

	return &r, nil
}

// -----------------
// Ubuntu
// -----------------

// mapDpkgInfoTest maps a DpkgInfoTestXML returning the test id along side the mapped DpkgInfoTest,
// will error out if the test id can not be parsed.
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

func mapUnixUnameTest(i oval_input.UnixUnameTestXML) (int, *oval_parsed.UnixUnameTest, error) {
	id, err := extractId(i.Id)
	if err != nil {
		return 0, nil, err
	}

	tst := oval_parsed.UnixUnameTest{}

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
		return nil, errors.New("only evr state definitions are supported")
	}

	r := oval_parsed.NewObjectStateEvrString(sta.Evr.Op, sta.Evr.Value)
	return &r, nil
}

func mapUnameState(sta oval_input.UnixUnameStateXML) *oval_parsed.ObjectStateString {
	r := oval_parsed.NewObjectStateString(sta.OSRelease.Op, sta.OSRelease.Value)
	return &r
}

func mapVariableTest(i oval_input.VariableTestXML) (int, *oval_parsed.UnixUnameTest, error) {
	id, err := extractId(i.Id)
	if err != nil {
		return 0, nil, err
	}

	tst := oval_parsed.UnixUnameTest{}

	return id, &tst, nil
}

func mapVariableState(sta oval_input.VariableStateXML) *oval_parsed.ObjectStateString {
	r := oval_parsed.NewObjectStateString(sta.Value.Op, sta.Value.Value)
	return &r
}
