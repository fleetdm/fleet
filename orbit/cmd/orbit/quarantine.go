package main

import (
	"net"
	"net/netip"
	"encoding/json"
	"os"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/tailscale/wf"
	"golang.org/x/sys/windows"
)

var _fleetUrl string

type QuarantineInfo struct {
	Sublayers	[]windows.GUID `json:"sublayers"`
	Rules		[]windows.GUID `json:"rules"`
}

func setFleetUrl(fleetUrl string) {
	temp := strings.Split(fleetUrl, "://")[1]
	temp  = strings.Split(temp, ":")[0]
	_fleetUrl = temp
	log.Debug().Msg("Quarantine: received fleet url: " + _fleetUrl)
}

func isQuarantined() bool {
	// TODO: check in windows registry
	if _, err := os.Stat(".\\I_am_quarantined"); err == nil {
		return true // File exists
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

func QuarantineIfNeeded() {
	if isQuarantined() {
		return
	}

	if runtime.GOOS != "windows" {
		// Only Windows is supported in this proof of concept
		log.Info().Msg("Quarantine is only supported on Windows currently")
		return
	}
	
	fwSession, err := wf.New(&wf.Options{
		Name:		"Quarantine Firewall Session",
		Dynamic:	false,
	})
	if err != nil {
		log.Error().Msg("Quarantine failed: Failed to start firewall session")
		return
	}

	fleetServerIPLookup, err := net.LookupIP(_fleetUrl)
	if err != nil {
		log.Error().Msg("Quarantine failed: Failed to find IP of fleet server: " +  _fleetUrl)
		log.Error().Msg(err.Error())
		return
	}
	var fleetServerIPv4 netip.Addr
	for _, ip := range fleetServerIPLookup {
		ipv4 := ip.To4()
		if ipv4 == nil {
			continue 
		}
		var ok bool
		fleetServerIPv4, ok = netip.AddrFromSlice(ipv4)
		if !ok {
			log.Error().Msg("Quarantine failed: Failed to convert server ip from LookupIP to netip.Addr: ")
			return
		}
	}
	
	// TODO: production quarantine should support IPv6 fleet server connection
	// Code to find both ipv4 and ipv6 addresses of the fleet server.
	/*var fleetServerIPv4, fleetServerIPv6 netip.Addr
	var foundIPv4, foundIPv6 bool
	for _, ip := range fleetServerIPLookup {
		if !foundIPv4 && ip.To4() != nil {
			address, ok := netip.AddrFromSlice(ip)
			if ok {
				fleetServerIPv4 = address
				foundIPv4 = true
			}
		}
		if !foundIPv6 && ip.To16() != nil && ip.To4() == nil {
			address, ok := netip.AddrFromSlice(ip)
			if ok {
				fleetServerIPv6 = address
				foundIPv6 = true
			}
		}
		if foundIPv4 && foundIPv6 {
			break
		}
	}
	if foundIPv4 {
		log.Debug().Msg("found fleet-url ipv4")
	}
	if foundIPv6 {
		log.Debug().Msg("found fleet-url ipv6")
	}*/
	
	addedRules := QuarantineInfo{make([]windows.GUID, 0), make([]windows.GUID, 0)}
	
	guidSublayer, err := windows.GenerateGUID()
	if err != nil {
		log.Error().Msg("Quarantine failed: Failed to generate windows GUID")
		return
	}
	sublayerID := wf.SublayerID(guidSublayer)
	err = fwSession.AddSublayer(&wf.Sublayer{
		ID:		sublayerID,
		Name:	"Quarantine killswitch",
		Weight:	0xffff,
	})
	if err != nil {
		log.Error().Msg("Quarantine failed: Failed to add sublayer")
		return
	}

	addedRules.Sublayers = append(addedRules.Sublayers, guidSublayer)

	layersIPv4 := []wf.LayerID{
		wf.LayerALEAuthRecvAcceptV4,
		wf.LayerALEAuthConnectV4,
	}
	
	layersIPv6 := []wf.LayerID{
		wf.LayerALEAuthRecvAcceptV6,
		wf.LayerALEAuthConnectV6,
	}

	/* Note: in production code, each rule that is added to the firewall 
	* 		 should have persistent set to true so that it remains after
	* 		 a system restart. For debug recoverability I have not set it. */
	
	for _, layer := range layersIPv4 {
		// Block all traffic except fleetServerIP
		guidBlockExceptFleet, err := windows.GenerateGUID()
		if err != nil {
			log.Error().Msg("Quarantine failed: Failed to generate windows GUID")
			return
		}
		addedRules.Rules = append(addedRules.Rules, guidBlockExceptFleet)
		err = fwSession.AddRule(&wf.Rule{
			ID:			wf.RuleID(guidBlockExceptFleet),
			Name:		"Block everything",
			Layer:		layer,
			Weight:		1000,
			Conditions:	[]*wf.Match{
				&wf.Match{
					Field:	wf.FieldIPRemoteAddress,
					Op:		wf.MatchTypeNotEqual,
					Value:	fleetServerIPv4,
				},
			},
			Action:		wf.ActionBlock,
			// Persistent: true
		})
		if err != nil {
			log.Error().Msg("Quarantine failed: Failed to add blocking rule")
			return
		}

		// NOTE: For the proof of concept I am allowing all traffic to port 53
		// to be able to keep the DNS server working, but it would be better to find
		// the DNS ip and only allow traffic to that
		guidDns, err := windows.GenerateGUID()
		if err != nil {
			log.Error().Msg("Quarantine failed: Failed to generate windows GUID")
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
			log.Error().Msg("Quarantine failed: Failed to add DNS port permit rule")
			return
		}

		// TODO: osquery needs its port to be allowed, otherwise Live Query from
		// the server does not work.
	}

	for _, layer := range layersIPv6 {
		 
		// TODO: production quarantine should support IPv6 fleet server connection
		// Code for allowing the fleet IPv6 address is here
		/*
		if foundIPv6 {
			// Allow traffic to the fleet server in case it is an ipv6 address
			guidAllowFleetIPv6, err := windows.GenerateGUID()
			if err != nil {
				log.Error().Msg("Quarantine failed: Failed to generate windows GUID")
				return
			}
			addedRules.Rules = append(addedRules.Rules, guidAllowFleetIPv6)
			err = fwSession.AddRule(&wf.Rule{
				ID:			wf.RuleID(guidAllowFleetIPv6),
				Name:		"Allow ipv6 fleet server",
				Layer:		layer,
				Weight:		800,
				Conditions:	[]*wf.Match{
					&wf.Match{
						Field:	wf.FieldIPRemoteAddress,
						Op:		wf.MatchTypeNotEqual,
						Value:	fleetServerIPv6,
					},
				},
				Action:		wf.ActionPermit,
				// Persistent: true
			})
			if err != nil {
				log.Error().Msg("Quarantine failed: Failed to add IPv6 fleet server allow rule")
				return
			}
		} */
		// Block all traffic except fleetServerIP
		guidBlockIPv6, err := windows.GenerateGUID()
		if err != nil {
			log.Error().Msg("Quarantine failed: Failed to generate windows GUID")
			return
		}
		addedRules.Rules = append(addedRules.Rules, guidBlockIPv6)
		err = fwSession.AddRule(&wf.Rule{
			ID:			wf.RuleID(guidBlockIPv6),
			Name:		"Block everything ipv6",
			Layer:		layer,
			Weight:		100,
			Conditions: nil,
			Action:		wf.ActionBlock,
			// Persistent: true
		})
		if err != nil {
			log.Error().Msg("Quarantine failed: Failed to add IPv6 blocking rule")
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
