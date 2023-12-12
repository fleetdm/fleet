package microsoft_mdm

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"io"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
)

// LoopHostMDMLocURIs loops all the <LocURI> values on all the profiles for a
// given host. It provides to the callback function:
//
// - An `ExpectedMDMProfile` that references the profile owning the LocURI
// - A hash that's unique for each profile/uri combination
// - The LocURI string
// - The data (if any) of the first <Item> element of the current LocURI
func LoopHostMDMLocURIs(
	ctx context.Context,
	ds fleet.ProfileVerificationStore,
	host *fleet.Host,
	fn func(profile *fleet.ExpectedMDMProfile, hash, locURI, data string),
) error {
	profileMap, err := ds.GetHostMDMProfilesExpectedForVerification(ctx, host)
	if err != nil {
		return fmt.Errorf("getting host profiles for verification: %w", err)
	}
	for _, expectedProf := range profileMap {
		var prof fleet.SyncMLCmd
		wrappedBytes := fmt.Sprintf("<Atomic>%s</Atomic>", expectedProf.RawProfile)
		if err := xml.Unmarshal([]byte(wrappedBytes), &prof); err != nil {
			return fmt.Errorf("unmarshalling profile %s: %w", expectedProf.Name, err)
		}
		for _, rc := range prof.ReplaceCommands {
			locURI := rc.GetTargetURI()
			data := rc.GetTargetData()
			ref := HashLocURI(expectedProf.Name, locURI)
			fn(expectedProf, ref, locURI, data)
		}
	}

	return nil
}

// HashLocURI creates a unique, consistent hash for a given profileName +
// locURI combination.
//
// FIXME: the mdm_bridge table decodes CmdID as `int`,
// so we encode the reference as an int32.
func HashLocURI(profileName, locURI string) string {
	hash := fnv.New32a()
	hash.Write([]byte(profileName + locURI))
	return fmt.Sprint(hash.Sum32())
}

// VerifyHostMDMProfiles performs the verification of the MDM profiles installed on a host and
// updates the verification status in the datastore. It is intended to be called by Fleet osquery
// service when the Fleet server ingests host details.
func VerifyHostMDMProfiles(ctx context.Context, ds fleet.ProfileVerificationStore, host *fleet.Host, rawSyncML []byte) error {
	var syncML fleet.SyncML
	decoder := xml.NewDecoder(bytes.NewReader(rawSyncML))
	// the DLL used by the `mdm_bridge` extension sends the response with
	// <?xml version="1.0" encoding="utf-16"?>, however if you use
	// `charset.NewReaderLabel` it fails to unmarshal (!?) for now, I'm
	// relying on this hack.
	decoder.CharsetReader = func(encoding string, input io.Reader) (io.Reader, error) {
		return input, nil
	}

	if err := decoder.Decode(&syncML); err != nil {
		return fmt.Errorf("decoding provided syncML: %w", err)
	}

	// TODO: what if more than one profile has the same
	// target uri but a different value? (product question)
	refToStatus := map[string]string{}
	refToResult := map[string]string{}
	for _, r := range syncML.GetOrderedCmds() {
		if r.Cmd.CmdRef == nil {
			continue
		}
		ref := *r.Cmd.CmdRef
		if r.Verb == fleet.CmdStatus && r.Cmd.Data != nil {
			refToStatus[ref] = *r.Cmd.Data
		}

		if r.Verb == fleet.CmdResults {
			refToResult[ref] = r.Cmd.GetTargetData()
		}
	}

	missing := map[string]struct{}{}
	verified := map[string]struct{}{}
	err := LoopHostMDMLocURIs(ctx, ds, host, func(profile *fleet.ExpectedMDMProfile, ref, locURI, wantData string) {
		// if we didn't get a status for a LocURI, mark the profile as
		// missing.
		gotStatus, ok := refToStatus[ref]
		if !ok {
			missing[profile.Name] = struct{}{}
		}
		// it's okay if we didn't get a result
		gotResults := refToResult[ref]
		// non-200 status don't have results. Consider it failed
		// TODO: should we be more granular instead? eg: special case
		// `4xx` responses? I'm sure there are edge cases we're not
		// accounting for here, but it's unclear at this moment.
		if !strings.HasPrefix(gotStatus, "2") || wantData != gotResults {
			withinGracePeriod := profile.IsWithinGracePeriod(host.DetailUpdatedAt)
			if !withinGracePeriod {
				missing[profile.Name] = struct{}{}
			}
			return
		}

		verified[profile.Name] = struct{}{}
	})
	if err != nil {
		return fmt.Errorf("looping host mdm LocURIs: %w", err)
	}

	toFail := make([]string, 0, len(missing))
	toRetry := make([]string, 0, len(missing))
	if len(missing) > 0 {
		counts, err := ds.GetHostMDMProfilesRetryCounts(ctx, host)
		if err != nil {
			return fmt.Errorf("getting host profiles retry counts: %w", err)
		}
		retriesByProfileUUID := make(map[string]uint, len(counts))
		for _, r := range counts {
			retriesByProfileUUID[r.ProfileName] = r.Retries
		}
		for key := range missing {
			// if the profile is in missing, we failed to validate at
			// least one LocURI, delete it from the verified map
			delete(verified, key)
			if retriesByProfileUUID[key] < mdm.MaxProfileRetries {
				// if we haven't hit the max retries, we set
				// the host profile status to nil (which causes
				// an install profile command to be enqueued
				// the next time the profile manager cron runs)
				// and increment the retry count
				toRetry = append(toRetry, key)
				continue
			}
			// otherwise we set the host profile status to failed
			toFail = append(toFail, key)
		}
	}

	i := 0
	verifiedSlice := make([]string, len(verified))
	for k := range verified {
		verifiedSlice[i] = k
		i++
	}
	return ds.UpdateHostMDMProfilesVerification(ctx, host, verifiedSlice, toFail, toRetry)
}
