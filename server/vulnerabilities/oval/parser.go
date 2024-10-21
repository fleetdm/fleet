package oval

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"

	oval_input "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/input"
	oval_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/parsed"
)

func parseDefinitions(platform Platform, inputFile string, outputFile string) error {
	r, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("oval parser: %w", err)
	}
	defer r.Close()

	var payload []byte
	switch {
	case platform.IsUbuntu():
		payload, err = processUbuntuDef(r)
	case platform.IsRedHat():
		payload, err = processRhelDef(r)
	}
	if err != nil {
		return fmt.Errorf("oval parser: %w", err)
	}

	err = os.WriteFile(outputFile, payload, 0o644)
	if err != nil {
		return fmt.Errorf("oval parser: %w", err)
	}

	return nil
}

// -----------------
// RHEL
// -----------------
func processRhelDef(r io.Reader) ([]byte, error) {
	xmlResult, err := parseRhelXML(r)
	if err != nil {
		return nil, err
	}

	result, err := mapToRhelResult(xmlResult)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func parseRhelXML(reader io.Reader) (*oval_input.RhelResultXML, error) {
	r := &oval_input.RhelResultXML{
		Variables: make(map[string]oval_input.ConstantVariableXML),
	}
	d := xml.NewDecoder(reader)

	for {
		t, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return r, nil
			}
			return nil, fmt.Errorf("decoding token: %v", err)
		}

		if t, ok := t.(xml.StartElement); ok {
			if t.Name.Local == "definition" {
				def := oval_input.DefinitionXML{}
				if err = d.DecodeElement(&def, &t); err != nil {
					return nil, err
				}
				r.Definitions = append(r.Definitions, def)
			}
			if t.Name.Local == "rpminfo_test" {
				tst := oval_input.RpmInfoTestXML{}
				if err = d.DecodeElement(&tst, &t); err != nil {
					return nil, err
				}
				r.RpmInfoTests = append(r.RpmInfoTests, tst)
			}
			if t.Name.Local == "rpmverifyfile_test" {
				tst := oval_input.RpmVerifyFileTestXML{}
				if err = d.DecodeElement(&tst, &t); err != nil {
					return nil, err
				}
				r.RpmVerifyFileTests = append(r.RpmVerifyFileTests, tst)
			}
			if t.Name.Local == "rpminfo_object" {
				sta := oval_input.PackageInfoTestObjectXML{}
				if err = d.DecodeElement(&sta, &t); err != nil {
					return nil, err
				}
				r.RpmInfoTestObjects = append(r.RpmInfoTestObjects, sta)
			}
			if t.Name.Local == "rpminfo_state" {
				obj := oval_input.RpmInfoStateXML{}
				if err = d.DecodeElement(&obj, &t); err != nil {
					return nil, err
				}
				r.RpmInfoTestStates = append(r.RpmInfoTestStates, obj)
			}
			if t.Name.Local == "rpmverifyfile_object" {
				obj := oval_input.RpmVerifyFileObjectXML{}
				if err = d.DecodeElement(&obj, &t); err != nil {
					return nil, err
				}
				r.RpmVerifyFileObjects = append(r.RpmVerifyFileObjects, obj)
			}
			if t.Name.Local == "rpmverifyfile_state" {
				sta := oval_input.RpmVerifyFileStateXML{}
				if err = d.DecodeElement(&sta, &t); err != nil {
					return nil, err
				}
				r.RpmVerifyFileStates = append(r.RpmVerifyFileStates, sta)
			}
			if t.Name.Local == "constant_variable" {
				cVar := oval_input.ConstantVariableXML{}
				if err = d.DecodeElement(&cVar, &t); err != nil {
					return nil, err
				}
				r.Variables[cVar.Id] = cVar
			}
		}
	}
}

func mapToRhelResult(xmlResult *oval_input.RhelResultXML) (*oval_parsed.RhelResult, error) {
	r := oval_parsed.NewRhelResult()

	rpmInfoObjToTst := make(map[string][]int)
	rpmInfoStaToTst := make(map[string][]int)

	rpmVerifyObjToTst := make(map[string][]int)
	rpmVerifyStaToTst := make(map[string][]int)

	for _, d := range xmlResult.Definitions {
		if len(d.Vulnerabilities) > 0 {
			def, err := mapDefinition(d)
			if err != nil {
				return nil, err
			}
			r.Definitions = append(r.Definitions, *def)
		}
	}

	// ------------
	// RpmInfoTests
	// ------------
	for _, t := range xmlResult.RpmInfoTests {
		id, tst, err := mapRpmInfoTest(t)
		if err != nil {
			return nil, err
		}

		rpmInfoObjToTst[t.Object.Id] = append(rpmInfoObjToTst[t.Object.Id], id)
		for _, sta := range t.States {
			rpmInfoStaToTst[sta.Id] = append(rpmInfoStaToTst[sta.Id], id)
		}
		r.RpmInfoTests[id] = tst
	}
	for _, o := range xmlResult.RpmInfoTestObjects {
		obj, err := mapPackageInfoTestObject(o, xmlResult.Variables)
		if err != nil {
			return nil, err
		}

		for _, tId := range rpmInfoObjToTst[o.Id] {
			t, ok := r.RpmInfoTests[tId]
			if ok {
				t.Objects = obj
			} else {
				return nil, fmt.Errorf("test not found: %d", tId)
			}
		}
	}
	for _, s := range xmlResult.RpmInfoTestStates {
		sta, err := mapRpmInfoState(s)
		if err != nil {
			return nil, err
		}
		for _, tId := range rpmInfoStaToTst[s.Id] {
			t, ok := r.RpmInfoTests[tId]
			if ok {
				t.States = append(t.States, *sta)
			} else {
				return nil, fmt.Errorf("test not found: %d", tId)
			}
		}
	}

	// ------------------
	// RpmVerifyFileTests
	// ------------------
	for _, t := range xmlResult.RpmVerifyFileTests {
		id, tst, err := mapRpmVerifyFileTest(t)
		if err != nil {
			return nil, err
		}
		rpmVerifyObjToTst[t.Object.Id] = append(rpmVerifyObjToTst[t.Object.Id], id)
		for _, sta := range t.States {
			rpmVerifyStaToTst[sta.Id] = append(rpmVerifyStaToTst[sta.Id], id)
		}
		r.RpmVerifyFileTests[id] = tst
	}
	for _, o := range xmlResult.RpmVerifyFileObjects {
		obj, err := mapRpmVerifyFileObject(o)
		if err != nil {
			return nil, err
		}

		for _, tId := range rpmVerifyObjToTst[o.Id] {
			t, ok := r.RpmVerifyFileTests[tId]
			if ok {
				t.FilePath = *obj
			} else {
				return nil, fmt.Errorf("test not found: %d", tId)
			}
		}
	}
	for _, s := range xmlResult.RpmVerifyFileStates {
		sta, err := mapRpmVerifyFileState(s)
		if err != nil {
			return nil, err
		}
		for _, tId := range rpmVerifyStaToTst[s.Id] {
			t, ok := r.RpmVerifyFileTests[tId]
			if ok {
				t.State = *sta
			} else {
				return nil, fmt.Errorf("test not found: %d", tId)
			}
		}
	}

	return r, nil
}

// -----------------
// Ubuntu
// -----------------

func processUbuntuDef(r io.Reader) ([]byte, error) {
	xmlResult, err := parseUbuntuXML(r)
	if err != nil {
		return nil, fmt.Errorf("parsing ubuntu xml: %w", err)
	}

	result, err := mapToUbuntuResult(xmlResult)
	if err != nil {
		return nil, fmt.Errorf("mapping ubuntu result: %w", err)
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshalling ubuntu result: %w", err)
	}

	return payload, nil
}

func parseUbuntuXML(reader io.Reader) (*oval_input.UbuntuResultXML, error) {
	r := &oval_input.UbuntuResultXML{
		Variables:       make(map[string]oval_input.ConstantVariableXML),
		VariableObjects: make(map[int]oval_input.VariableObjectXML),
	}
	d := xml.NewDecoder(reader)

	for {
		t, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return r, nil
			}
			return nil, fmt.Errorf("decoding token: %v", err)
		}

		if t, ok := t.(xml.StartElement); ok {
			if t.Name.Local == "definition" {
				def := oval_input.DefinitionXML{}
				if err = d.DecodeElement(&def, &t); err != nil {
					return nil, fmt.Errorf("decoding definition: %w", err)
				}
				r.Definitions = append(r.Definitions, def)
			}
			if t.Name.Local == "dpkginfo_test" {
				tst := oval_input.DpkgInfoTestXML{}
				if err = d.DecodeElement(&tst, &t); err != nil {
					return nil, fmt.Errorf("decoding dpkginfo_test: %w", err)
				}
				r.DpkgInfoTests = append(r.DpkgInfoTests, tst)
			}
			if t.Name.Local == "dpkginfo_state" {
				sta := oval_input.DpkgInfoStateXML{}
				if err = d.DecodeElement(&sta, &t); err != nil {
					return nil, fmt.Errorf("decoding dpkginfo_state: %w", err)
				}
				r.DpkgInfoStates = append(r.DpkgInfoStates, sta)
			}
			if t.Name.Local == "dpkginfo_object" {
				obj := oval_input.PackageInfoTestObjectXML{}
				if err = d.DecodeElement(&obj, &t); err != nil {
					return nil, fmt.Errorf("decoding dpkginfo_object: %w", err)
				}
				r.DpkgInfoObjects = append(r.DpkgInfoObjects, obj)
			}
			if t.Name.Local == "uname_test" {
				tst := oval_input.UnixUnameTestXML{}
				if err = d.DecodeElement(&tst, &t); err != nil {
					return nil, fmt.Errorf("decoding uname_test: %w", err)
				}
				r.UnameTests = append(r.UnameTests, tst)
			}
			if t.Name.Local == "uname_state" {
				sta := oval_input.UnixUnameStateXML{}
				if err = d.DecodeElement(&sta, &t); err != nil {
					return nil, fmt.Errorf("decoding uname_state: %w", err)
				}
				r.UnameStates = append(r.UnameStates, sta)
			}
			if t.Name.Local == "variable_test" {
				tst := oval_input.VariableTestXML{}
				if err = d.DecodeElement(&tst, &t); err != nil {
					return nil, fmt.Errorf("decoding variable_test: %w", err)
				}
				r.VariableTests = append(r.VariableTests, tst)
			}
			if t.Name.Local == "variable_object" {
				obj := oval_input.VariableObjectXML{}
				if err = d.DecodeElement(&obj, &t); err != nil {
					return nil, fmt.Errorf("decoding variable_object: %w", err)
				}

				id, err := extractId(obj.Id)
				if err != nil {
					return nil, fmt.Errorf("extracting id: %w", err)
				}
				r.VariableObjects[id] = obj
			}

			if t.Name.Local == "variable_state" {
				sta := oval_input.VariableStateXML{}
				if err = d.DecodeElement(&sta, &t); err != nil {
					return nil, fmt.Errorf("decoding variable_state: %w", err)
				}
				r.VariableStates = append(r.VariableStates, sta)
			}
			if t.Name.Local == "constant_variable" {
				cVar := oval_input.ConstantVariableXML{}
				if err = d.DecodeElement(&cVar, &t); err != nil {
					return nil, fmt.Errorf("decoding constant_variable: %w", err)
				}
				r.Variables[cVar.Id] = cVar
			}
		}
	}
}

func mapToUbuntuResult(xmlResult *oval_input.UbuntuResultXML) (*oval_parsed.UbuntuResult, error) {
	r := oval_parsed.NewUbuntuResult()

	staToTst := make(map[string][]int)
	objToTst := make(map[string][]int)
	ustaToTst := make(map[string][]int)
	vstaToTst := make(map[string][]int)

	for _, d := range xmlResult.Definitions {
		if len(d.Vulnerabilities) > 0 {
			def, err := mapDefinition(d)
			if err != nil {
				return nil, fmt.Errorf("mapping definition: %w", err)
			}
			r.AddDefinition(*def)
		}
	}

	for _, t := range xmlResult.DpkgInfoTests {
		id, tst, err := mapDpkgInfoTest(t)
		if err != nil {
			return nil, fmt.Errorf("mapping dpkg info test: %w", err)
		}

		objToTst[t.Object.Id] = append(objToTst[t.Object.Id], id)
		for _, sta := range t.States {
			staToTst[sta.Id] = append(staToTst[sta.Id], id)
		}
		r.AddPackageTest(id, tst)
	}

	for _, o := range xmlResult.DpkgInfoObjects {
		obj, err := mapPackageInfoTestObject(o, xmlResult.Variables)
		if err != nil {
			return nil, fmt.Errorf("mapping dpkg info object: %w", err)
		}

		for _, tId := range objToTst[o.Id] {
			t, ok := r.PackageTests[tId]
			if ok {
				t.Objects = obj
			} else {
				return nil, fmt.Errorf("test not found: %d", tId)
			}
		}
	}

	for _, s := range xmlResult.DpkgInfoStates {
		sta, err := mapDpkgInfoState(s)
		if err != nil {
			return nil, fmt.Errorf("mapping dpkg info state: %w", err)
		}
		for _, tId := range staToTst[s.Id] {
			t, ok := r.PackageTests[tId]
			if ok {
				t.States = append(t.States, *sta)
			} else {
				return nil, fmt.Errorf("test not found: %d", tId)
			}
		}
	}

	for _, t := range xmlResult.UnameTests {
		id, tst, err := mapUnixUnameTest(t)
		if err != nil {
			return nil, fmt.Errorf("mapping uname test: %w", err)
		}

		for _, sta := range t.States {
			ustaToTst[sta.Id] = append(ustaToTst[sta.Id], id)
		}
		r.AddUnameTest(id, tst)
	}

	for _, s := range xmlResult.UnameStates {
		sta := mapUnameState(s)
		for _, tId := range ustaToTst[s.Id] {
			t, ok := r.UnameTests[tId]
			if ok {
				t.States = append(t.States, *sta)
			} else {
				return nil, fmt.Errorf("test not found: %d", tId)
			}
		}
	}

	for _, t := range xmlResult.VariableTests {
		id, tst, err := mapVariableTest(t)
		if err != nil {
			return nil, fmt.Errorf("mapping variable test: %w", err)
		}

		// Skip tests that are not used in any uname test
		// (there should be none)
		if obj, ok := xmlResult.VariableObjects[id]; ok {
			id, err := extractId(obj.RefID)
			if err != nil {
				return nil, fmt.Errorf("extracting variable object id: %w", err)
			}
			if _, ok := r.UnameTests[id]; !ok {
				continue
			}
		}

		for _, sta := range t.States {
			vstaToTst[sta.Id] = append(vstaToTst[sta.Id], id)
		}
		r.AddUnameTest(id, tst)
	}

	for _, s := range xmlResult.VariableStates {
		sta := mapVariableState(s)
		for _, tId := range vstaToTst[s.Id] {
			t, ok := r.UnameTests[tId]
			if ok {
				t.States = append(t.States, *sta)
			} else {
				return nil, fmt.Errorf("test not found: %d", tId)
			}
		}
	}

	return r, nil
}
