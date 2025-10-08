package fleet

import (
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// install/removeTargets are maps from profileUUID -> command uuid and host
// UUIDs as the underlying MDM services are optimized to send one command to
// multiple hosts at the same time. Note that the same command uuid is used
// for all hosts in a given install/remove target operation.
type CmdTarget struct {
	CmdUUID           string
	ProfileIdentifier string
	EnrollmentIDs     []string
}

type HostProfileUUID struct {
	HostUUID    string
	ProfileUUID string
}

func FindProfilesWithSecrets(
	logger kitlog.Logger,
	installTargets map[string]*CmdTarget,
	profileContents map[string]mobileconfig.Mobileconfig,
) (map[string]struct{}, error) {
	profilesWithSecrets := make(map[string]struct{})
	for profUUID := range installTargets {
		p, ok := profileContents[profUUID]
		if !ok { // Should never happen
			level.Error(logger).Log("msg", "profile content not found in ReconcileAppleProfiles", "profile_uuid", profUUID)
			continue
		}
		profileStr := string(p)
		vars := ContainsPrefixVars(profileStr, ServerSecretPrefix)
		if len(vars) > 0 {
			profilesWithSecrets[profUUID] = struct{}{}
		}
	}
	return profilesWithSecrets, nil
}
