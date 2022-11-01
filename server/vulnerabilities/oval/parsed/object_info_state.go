package oval_parsed

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
)

type ObjectInfoState struct {
	Name           *ObjectStateString      `json:",omitempty"`
	Arch           *ObjectStateString      `json:",omitempty"`
	Epoch          *ObjectStateSimpleValue `json:",omitempty"`
	Release        *ObjectStateSimpleValue `json:",omitempty"`
	Version        *ObjectStateSimpleValue `json:",omitempty"`
	Evr            *ObjectStateEvrString   `json:",omitempty"`
	SignatureKeyId *ObjectStateString      `json:",omitempty"`
	ExtendedName   *ObjectStateString      `json:",omitempty"`
	FilePath       *ObjectStateString      `json:",omitempty"`
	Operator       OperatorType            `json:"operator"`
}

// EvalSoftware evaluates the software against the specified state.
func (sta ObjectInfoState) EvalSoftware(s fleet.Software) (bool, error) {
	var results []bool

	if sta.Name != nil {
		rEval, err := sta.Name.Eval(s.Name)
		if err != nil {
			return false, err
		}
		results = append(results, rEval)
	}

	if sta.Arch != nil {
		rEval, err := sta.Arch.Eval(s.Arch)
		if err != nil {
			return false, err
		}
		results = append(results, rEval)
	}

	// TODO: see https://github.com/fleetdm/fleet/issues/6236 -
	// For RHEL based systems the epoch is not included in the version field
	// if sta.Epoch != nil {
	// 	rEval, err := sta.Epoch.Eval(fmt.Sprint(epoch(s.Version)))
	// 	if err != nil {
	// 		return false, err
	// 	}
	// 	results = append(results, rEval)
	// }

	if sta.Release != nil {
		var rel string
		if s.Release != "" {
			// Check if the software has a release
			rel = s.Release
		} else {
			// If not, try to get it from the version
			rel = utils.Release(s.Version)
		}
		rEval, err := sta.Release.Eval(rel)
		if err != nil {
			return false, err
		}
		results = append(results, rEval)
	}

	if sta.Version != nil {
		rEval, err := sta.Version.Eval(s.Version)
		if err != nil {
			return false, err
		}
		results = append(results, rEval)
	}

	if sta.Evr != nil {
		var evr string
		if s.Release != "" {
			// If the release is set, append it to version
			evr = fmt.Sprintf("%s-%s", s.Version, s.Release)
		} else {
			evr = s.Version
		}

		// TODO: see https://github.com/fleetdm/fleet/issues/6236 -
		// ATM we are not storing the epoch, so we will need to removed it from the
		// state ... otherwise we will
		rEval, err := sta.Evr.Eval(evr, utils.Rpmvercmp, true)
		if err != nil {
			return false, err
		}
		results = append(results, rEval)
	}

	if sta.SignatureKeyId != nil {
		// Assume that all installed software was signed by the proper third party (RedHat), we are
		// doing this basically because there's no way to get the signature key ATM and even if we
		// have it we want to reuse the RHEL OVAL definitions for CentOS
		results = append(results, true)
	}

	if len(results) == 0 {
		return false, errors.New("invalid empty state")
	}

	return sta.Operator.Eval(results...), nil
}

func (sta ObjectInfoState) EvalOSVersion(version fleet.OSVersion) (bool, error) {
	var results []bool

	// If 'sta' is used for specifying the state of a RpmVerifyFile test, 'Name' refers to the name of the
	// file, when making assertions against the installed OS, the file in question will be
	// /etc/redhat-release, so in order to use the same test for CentOS distros, we will need to
	// normalize the value.
	if sta.Name != nil {
		var nName string
		if version.Platform == "rhel" || version.Platform == "amzn" {
			nName = "redhat-release"
		}
		rEval, err := sta.Name.Eval(nName)
		if err != nil {
			return false, err
		}
		results = append(results, rEval)
	}

	if sta.Version != nil {
		var pVer string
		if version.Platform == "rhel" {
			version := ReplaceFedoraOSVersion(version.Name)
			pName := strings.Trim(version, " ")
			pVer = pName[strings.LastIndex(pName, " ")+1:]
		}

		if version.Platform == "amzn" {
			// Amazon Linux 2 is based on RHEL 7
			pVer = "7.0.0"
		}

		rEval, err := sta.Version.Eval(pVer)
		if err != nil {
			return false, err
		}
		results = append(results, rEval)
	}

	if len(results) == 0 {
		return false, errors.New("invalid empty state")
	}

	return sta.Operator.Eval(results...), nil
}
