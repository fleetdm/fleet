package android

import "encoding/json"

// ONC structs. Minimal types for extracting certificate alias references from
// Android's openNetworkConfiguration policy field (Chrome OS ONC spec).

type oncConfig struct {
	NetworkConfigurations []oncNetworkConfiguration `json:"NetworkConfigurations"`
}

type oncNetworkConfiguration struct {
	WiFi     *oncEAPWrapper    `json:"WiFi,omitempty"`
	Ethernet *oncEAPWrapper    `json:"Ethernet,omitempty"`
	VPN      *oncCertKeyHolder `json:"VPN,omitempty"`
}

// oncEAPWrapper is shared by WiFi and Ethernet, which nest the cert alias inside an EAP sub-object.
type oncEAPWrapper struct {
	EAP *oncCertKeyHolder `json:"EAP,omitempty"`
}

// oncCertKeyHolder holds certificate client auth fields. ClientCertKeyPairAlias is only
// meaningful when ClientCertType is "KeyPairAlias" per the ONC spec; otherwise it is ignored.
type oncCertKeyHolder struct {
	ClientCertType         string `json:"ClientCertType,omitempty"`
	ClientCertKeyPairAlias string `json:"ClientCertKeyPairAlias,omitempty"`
}

// extractAlias returns the ClientCertKeyPairAlias only when ClientCertType is "KeyPairAlias".
// Per the ONC spec, the alias field is ignored for all other ClientCertType values.
func extractAlias(h *oncCertKeyHolder) string {
	if h.ClientCertType == "KeyPairAlias" && h.ClientCertKeyPairAlias != "" {
		return h.ClientCertKeyPairAlias
	}
	return ""
}

// ExtractCertAliasesFromONC parses an openNetworkConfiguration JSON blob
// and returns all ClientCertKeyPairAlias values found.
func ExtractCertAliasesFromONC(oncJSON json.RawMessage) ([]string, error) {
	var onc oncConfig
	if err := json.Unmarshal(oncJSON, &onc); err != nil {
		return nil, err
	}

	var aliases []string
	for _, nc := range onc.NetworkConfigurations {
		if nc.WiFi != nil && nc.WiFi.EAP != nil {
			if a := extractAlias(nc.WiFi.EAP); a != "" {
				aliases = append(aliases, a)
			}
		}
		if nc.Ethernet != nil && nc.Ethernet.EAP != nil {
			if a := extractAlias(nc.Ethernet.EAP); a != "" {
				aliases = append(aliases, a)
			}
		}
		if nc.VPN != nil {
			if a := extractAlias(nc.VPN); a != "" {
				aliases = append(aliases, a)
			}
		}
	}
	return aliases, nil
}

// ExtractCertAliasesFromProfileJSON parses an Android profile's raw JSON
// and returns any ClientCertKeyPairAlias values found in its
// openNetworkConfiguration field. Returns nil if no ONC field exists.
func ExtractCertAliasesFromProfileJSON(profileJSON json.RawMessage) ([]string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(profileJSON, &fields); err != nil {
		return nil, err
	}
	oncJSON, hasONC := fields["openNetworkConfiguration"]
	if !hasONC {
		return nil, nil
	}
	return ExtractCertAliasesFromONC(oncJSON)
}
