// Package admx handles ADMX (Administrative Template File) policies for Microsoft MDM server.
// See: https://learn.microsoft.com/en-us/windows/client-management/understanding-admx-backed-policies
//
// ADMX policy payload example:
// <![CDATA[
//
//	<enabled/>
//	<data id="Publishing_Server2_Name_Prompt" value="Name"/>
//	<data id="Publishing_Server_URL_Prompt" value="http://someuri"/>
//	<data id="Global_Publishing_Refresh_Options" value="1"/>
//
// ]]>
package admx

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"regexp"
	"slices"
	"strings"
)

var cdataRegexp = regexp.MustCompile(`(?s)<!\[CDATA\[(.*?)]]>`)

func IsADMX(text string) bool {
	// We try to unmarshal the string to see if it looks like a valid ADMX policy
	policy, err := unmarshal(text)
	if err != nil {
		return false
	}
	return policy.Enabled.Local == "enabled" || policy.Disabled.Local == "disabled" || len(policy.Data) > 0
}

func Equal(a, b string) (bool, error) {
	aPolicy, err := unmarshal(a)
	if err != nil {
		return false, fmt.Errorf("unmarshalling ADMX policy a: %w", err)
	}
	bPolicy, err := unmarshal(b)
	if err != nil {
		return false, fmt.Errorf("unmarshalling ADMX policy b: %w", err)
	}
	return aPolicy.Equal(bPolicy), nil
}

func unmarshal(a string) (admxPolicy, error) {
	a, err := convertToXMLString(a)
	if err != nil {
		return admxPolicy{}, fmt.Errorf("converting ADMX policy to XML string: %w", err)
	}
	var policy admxPolicy
	err = xml.NewDecoder(bytes.NewReader([]byte(a))).Decode(&policy)
	if err != nil {
		return admxPolicy{}, fmt.Errorf("unmarshalling ADMX policy: %w", err)
	}
	return policy, nil
}

func convertToXMLString(a string) (string, error) {
	matches := cdataRegexp.FindAllStringSubmatch(a, -1)
	if len(matches) > 1 {
		return "", errors.New("multiple CDATA matches found in ADMX policy")
	}
	if len(matches) == 1 && len(matches[0]) > 1 {
		// If CDATA is present, we extract the content. Otherwise, we use the original string.
		a = matches[0][1]
	}
	a = html.UnescapeString(a)
	// ADMX policy elements are not case-sensitive. For example: <enabled/> and <Enabled/> are equivalent
	// For simplicity, we compare everything in lowercase.
	a = strings.ToLower(a)
	// We wrap the policy in a <policy> tag to ensure it can be unmarshalled by the XML decoder
	a = `<policy>` + a + `</policy>`
	return a, nil
}

type admxPolicy struct {
	Enabled  xml.Name         `xml:"enabled,omitempty"`
	Disabled xml.Name         `xml:"disabled,omitempty"`
	Data     []admxPolicyItem `xml:"data"`
}

func (a admxPolicy) Equal(b admxPolicy) bool {
	if a.Disabled.Local != b.Disabled.Local {
		return false
	}
	if a.Disabled.Local == "disabled" {
		// If the ADMX policy is disabled, the data is not relevant
		return true
	}
	if a.Enabled.Local != b.Enabled.Local {
		return false
	}
	if len(a.Data) != len(b.Data) {
		return false
	}
	a.sortData()
	b.sortData()
	for i := range a.Data {
		if !a.Data[i].Equal(b.Data[i]) {
			return false
		}
	}
	return true
}

func (a *admxPolicy) sortData() {
	slices.SortFunc(a.Data, func(i, j admxPolicyItem) int {
		return strings.Compare(i.ID, j.ID)
	})
}

type admxPolicyItem struct {
	ID    string `xml:"id,attr"`
	Value string `xml:"value,attr"`
}

func (a admxPolicyItem) Equal(b admxPolicyItem) bool {
	return a.ID == b.ID && a.Value == b.Value
}
