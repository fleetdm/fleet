package oval

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
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

	err = ioutil.WriteFile(outputFile, payload, 0o644)
	if err != nil {
		return fmt.Errorf("oval parser: %w", err)
	}

	return nil
}

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

func parseRhelXML(r io.Reader) (*oval_input.RhelResultXML, error) {
	panic("not implemented")
}

func mapToRhelResult(xmlResult *oval_input.RhelResultXML) (*oval_parsed.RhelResult, error) {
	panic("not implemented")
}

func processUbuntuDef(r io.Reader) ([]byte, error) {
	xmlResult, err := parseUbuntuXML(r)
	if err != nil {
		return nil, err
	}

	result, err := mapToUbuntuResult(xmlResult)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func parseUbuntuXML(reader io.Reader) (*oval_input.UbuntuResultXML, error) {
	r := &oval_input.UbuntuResultXML{
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

		switch t := t.(type) {
		case xml.StartElement:
			if t.Name.Local == "definition" {
				def := oval_input.DefinitionXML{}
				if err = d.DecodeElement(&def, &t); err != nil {
					return nil, err
				}
				r.Definitions = append(r.Definitions, def)
			}

			if t.Name.Local == "dpkginfo_test" {
				tst := oval_input.DpkgInfoTestXML{}
				if err = d.DecodeElement(&tst, &t); err != nil {
					return nil, err
				}
				r.PackageTests = append(r.PackageTests, tst)
			}
			if t.Name.Local == "dpkginfo_state" {
				sta := oval_input.DpkgInfoStateXML{}
				if err = d.DecodeElement(&sta, &t); err != nil {
					return nil, err
				}
				r.PackageStates = append(r.PackageStates, sta)
			}
			if t.Name.Local == "dpkginfo_object" {
				obj := oval_input.DpkgInfoObjectXML{}
				if err = d.DecodeElement(&obj, &t); err != nil {
					return nil, err
				}
				r.PackageObjects = append(r.PackageObjects, obj)

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

func mapToUbuntuResult(xmlResult *oval_input.UbuntuResultXML) (*oval_parsed.UbuntuResult, error) {
	r := oval_parsed.NewUbuntuResult()

	staToTst := make(map[string][]int)
	objToTst := make(map[string][]int)

	for _, d := range xmlResult.Definitions {
		if len(d.CVEs) > 0 {
			def, err := mapDefinition(d)
			if err != nil {
				return nil, err
			}
			r.AddDefinition(*def)
		}
	}

	for _, t := range xmlResult.PackageTests {
		id, tst, err := mapPackageTest(t)
		if err != nil {
			return nil, err
		}

		objToTst[t.Object.Id] = append(objToTst[t.Object.Id], id)
		for _, sta := range t.States {
			staToTst[sta.Id] = append(staToTst[sta.Id], id)
		}

		r.AddPackageTest(id, tst)
	}

	for _, o := range xmlResult.PackageObjects {
		obj, err := mapPackageObject(o, xmlResult.Variables)
		if err != nil {
			return nil, err
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

	for _, s := range xmlResult.PackageStates {
		sta, err := mapPackageState(s)
		if err != nil {
			return nil, err
		}

		for _, tId := range staToTst[s.Id] {
			t, ok := r.PackageTests[tId]
			if ok {
				t.States = sta
			} else {
				return nil, fmt.Errorf("test not found: %d", tId)
			}
		}
	}
	return r, nil
}
