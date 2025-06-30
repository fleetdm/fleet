package main

import (
	"net"
	"net/netip"
	//"net/netip"
	"encoding/json"
	"os"
	"runtime"

	"github.com/rs/zerolog/log"
	"github.com/tailscale/wf"
	"golang.org/x/sys/windows"
)

type QuarantineInfo struct {
	Sublayers	[]windows.GUID `json:"sublayers"`
	Rules		[]windows.GUID `json:"rules"`
}

func isQuarantined() bool {
	// TODO: check in windows registry
	if _, err := os.Stat(".\\I_am_quarantined"); err == nil {
		// File exists
		return true
	}
	return false
}

func markQuarantined() {
	// TODO: save this information in windows registry
	os.Create(".\\I_am_quarantined")
}

func markUnquarantined() {
	// TODO: remove this information in windows registry
	os.Remove(".\\I_am_quarantined")
}

func QuarantineIfNeeded(fleetUrl string) {
	if isQuarantined() {
		return
	}

	if runtime.GOOS != "windows" {
		// Only Windows is supported
		log.Info().Msg("Quarantine is only supported on Windows currently")
		return
	}
	
	fwSession, err := wf.New(&wf.Options{
		Name:		"Quarantine Firewall Session",
		Dynamic:	false,
	})
	if err != nil {
		log.Error().Msg("Quarantine failed: Failed to start firewall session: ")
		return
	}

	fleetServerIPLookup, err := net.LookupIP(fleetUrl) // ipv4
	if err != nil {
		log.Error().Msg("Quarantine failed: Failed find IP of fleet server: ")
		return
	}
	var fleetServerIP netip.Addr
	// Convert []Ip to Addr
	for _, ip := range fleetServerIPLookup {
		ipv4 := ip.To4()
		if ipv4 == nil {
			continue 
		}
		var ok bool
		fleetServerIP, ok = netip.AddrFromSlice(ipv4)
		if !ok {
			log.Error().Msg("Quarantine failed: Failed to convert server ip from LookupIP to netip.Addr: ")
			return
		}
	}
	
	addedRules := QuarantineInfo{make([]windows.GUID, 0), make([]windows.GUID, 0)}
	
	guidSublayer, err := windows.GenerateGUID()
	if err != nil {
		log.Error().Msg("Quarantine failed: Failed to generate windows GUID: ")
		return
	}
	sublayerID := wf.SublayerID(guidSublayer)
	err = fwSession.AddSublayer(&wf.Sublayer{
		ID:		sublayerID,
		Name:	"Quarantine killswitch",
		Weight:	0xffff,
	})
	if err != nil {
		log.Error().Msg("Quarantine failed: Failed to add sublayer: ")
		return
	}
	addedRules.Sublayers = append(addedRules.Sublayers, guidSublayer)

	// TODO: Add ipv6 version of quarantine
	layers := []wf.LayerID{
		wf.LayerALEAuthRecvAcceptV4,
		//wf.LayerALEAuthRecvAcceptV6,
		wf.LayerALEAuthConnectV4,
		//wf.LayerALEAuthConnectV6,
	}
	
	for _, layer := range layers {
		// Block all traffic except fleetServerIP
		guidBlock, err := windows.GenerateGUID()
		if err != nil {
			log.Error().Msg("Quarantine failed: Failed to generate windows GUID: ")
			return
		}
		addedRules.Rules = append(addedRules.Rules, guidBlock)
		err = fwSession.AddRule(&wf.Rule{
			ID:			wf.RuleID(guidBlock),
			Name:		"Block everything",
			Layer:		layer,
			Weight:		100,
			Conditions:	[]*wf.Match{
				&wf.Match{
					Field:	wf.FieldIPRemoteAddress,
					Op:		wf.MatchTypeNotEqual,
					Value:	fleetServerIP,
				},
			},
			Action:		wf.ActionBlock,
			// Persistent: true
		})
		if err != nil {
			log.Error().Msg("Quarantine failed: Failed to add blocking rule: ")
			return
		}

		// NOTE: For the proof of concept I am allowing all traffic to port 53
		// to be able to keep the DNS server working, but this is UNSAFE
		guidDns, err := windows.GenerateGUID()
		if err != nil {
			log.Error().Msg("Quarantine failed: Failed to generate windows GUID: ")
			return
		}
		addedRules.Rules = append(addedRules.Rules, guidDns)
		err = fwSession.AddRule(&wf.Rule{
			ID:			wf.RuleID(guidDns),
			Name:		"Allow DNS port",
			Layer:		layer,
			Weight:		900,
			Conditions:	[]*wf.Match{
				&wf.Match{
					Field:	wf.FieldIPRemotePort,
					Op:		wf.MatchTypeEqual,
					Value:	uint16(53),
				},
			},
			Action:		wf.ActionPermit,
			// Persistent: true
		})
		if err != nil {
			log.Error().Msg("Quarantine failed: Failed to add DNS port permit rule: ")
			return
		}
	}

	SaveAllCustomRules(&addedRules)
	markQuarantined()
	fwSession.Close()
}
func UnquarantineIfNeeded() {
	if !isQuarantined() {
		return
	}
	fwSession, err := wf.New(&wf.Options{
		Name:		"UnQuarantine WPF session",
		Dynamic:	false,
	})
	if err != nil {
		log.Error().Msg("Unquarantine failed: Failed to start WFP session: ")
	}
	rules := LoadAllCustomRules()
	RemoveAllCustomRules(fwSession, rules)
	markUnquarantined()
}

func PrintAllCustomRules(customRules QuarantineInfo) {
	//log.Println("Subalyers:")
	for _, id := range customRules.Sublayers {
		log.Debug().Msg(id.String())
	}
	//log.Println("Rules:")
	for _, id := range customRules.Rules {
		log.Debug().Msg(id.String())
	}
}

func RemoveAllCustomRules(fwSession *wf.Session, customRules QuarantineInfo) {
	//log.Println("Remove all custom rules")
	for _, id := range customRules.Sublayers {
		fwSession.DeleteSublayer(wf.SublayerID(id))
	}
	for _, id := range customRules.Rules {
		fwSession.DeleteRule(wf.RuleID(id))
	}
}

func SaveAllCustomRules(customRules *QuarantineInfo) {
	ruleFile, err := os.Create(".\\quarantine_rules.json")
	if err != nil {
		log.Error().Msg("Quarantine failed: Failed to create quarantine_rules.json: ")
		return
	}
	enc := json.NewEncoder(ruleFile)
	err = enc.Encode(customRules)
	if err != nil {
		log.Error().Msg("Quarantine failed: Error encoding json: ")
		return
	}
}

func LoadAllCustomRules() (QuarantineInfo) {
	ruleFile, err := os.Open(".\\quarantine_rules.json")
	if err != nil {
		log.Error().Msg("Quarantine failed: Error opening file quarantine_rules.json: ")
		return QuarantineInfo{}
	}
	data := QuarantineInfo{make([]windows.GUID, 0), make([]windows.GUID, 0)}
	dec := json.NewDecoder(ruleFile)
	err = dec.Decode(&data)
	return data
}
