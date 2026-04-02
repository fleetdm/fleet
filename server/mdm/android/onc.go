package android

import "encoding/json"

// ONC structs - minimal types for extracting certificate alias references from
// Android's openNetworkConfiguration policy field (Chrome OS ONC spec).

type oncConfig struct {
	NetworkConfigurations []oncNetworkConfiguration `json:"NetworkConfigurations"`
}

type oncNetworkConfiguration struct {
	WiFi     *oncWiFi     `json:"WiFi,omitempty"`
	Ethernet *oncEthernet `json:"Ethernet,omitempty"`
	VPN      *oncVPN      `json:"VPN,omitempty"`
}

type oncWiFi struct {
	EAP *oncEAP `json:"EAP,omitempty"`
}

type oncEthernet struct {
	EAP *oncEAP `json:"EAP,omitempty"`
}

type oncEAP struct {
	ClientCertKeyPairAlias string `json:"ClientCertKeyPairAlias,omitempty"`
}

type oncVPN struct {
	ClientCertKeyPairAlias string `json:"ClientCertKeyPairAlias,omitempty"`
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
		if nc.WiFi != nil && nc.WiFi.EAP != nil && nc.WiFi.EAP.ClientCertKeyPairAlias != "" {
			aliases = append(aliases, nc.WiFi.EAP.ClientCertKeyPairAlias)
		}
		if nc.Ethernet != nil && nc.Ethernet.EAP != nil && nc.Ethernet.EAP.ClientCertKeyPairAlias != "" {
			aliases = append(aliases, nc.Ethernet.EAP.ClientCertKeyPairAlias)
		}
		if nc.VPN != nil && nc.VPN.ClientCertKeyPairAlias != "" {
			aliases = append(aliases, nc.VPN.ClientCertKeyPairAlias)
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
