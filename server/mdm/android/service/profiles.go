package service

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	kitlog "github.com/go-kit/log"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

func ReconcileProfiles(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, licenseKey string) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get app config")
	}
	if !appConfig.MDM.AndroidEnabledAndConfigured {
		return nil
	}

	// get the one-and-only Android enterprise, which is treated as an error if
	// not present, since the appconfig tells us Android MDM is enabled and
	// configured.
	enterprise, err := ds.GetEnterprise(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android enterprise")
	}

	client := newAMAPIClient(ctx, logger, licenseKey)
	authSecret, err := getClientAuthenticationSecret(ctx, ds)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting Android client authentication secret for profile reconciler")
	}
	err = client.SetAuthenticationSecret(authSecret)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "setting Android client authentication secret for profile reconciler")
	}

	reconciler := &profileReconciler{
		DS:         ds,
		Enterprise: enterprise,
		Client:     client,
	}
	return reconciler.ReconcileProfiles(ctx)
}

// profileReconciler is a struct to facilitate testability, it should not be
// used outside of tests.
type profileReconciler struct {
	DS         fleet.Datastore
	Enterprise *android.Enterprise
	Client     androidmgmt.Client
}

func getClientAuthenticationSecret(ctx context.Context, ds fleet.Datastore) (string, error) {
	assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidFleetServerSecret}, nil)
	switch {
	case fleet.IsNotFound(err):
		return "", nil
	case err != nil:
		return "", ctxerr.Wrap(ctx, err, "getting Android authentication secret")
	}
	return string(assets[fleet.MDMAssetAndroidFleetServerSecret].Value), nil
}

func (r *profileReconciler) ReconcileProfiles(ctx context.Context) error {
	if err := r.reconcileCertificateTemplates(ctx); err != nil {
		return ctxerr.Wrap(ctx, err, "reconcile certificate templates")
	}

	// get the list of hosts that need to have their profiles applied
	hostsApplicableProfiles, hostsProfsToRemove, err := r.DS.ListMDMAndroidProfilesToSend(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "identify android profiles to send")
	}

	profilesByHostUUID := make(map[string][]*fleet.MDMAndroidProfilePayload)
	profilesToLoad := make(map[string]struct{})
	for _, hostProf := range hostsApplicableProfiles {
		profilesByHostUUID[hostProf.HostUUID] = append(profilesByHostUUID[hostProf.HostUUID], hostProf)

		// keep a deduplicated list of profiles to load the JSON only once for each
		// distinct one
		profilesToLoad[hostProf.ProfileUUID] = struct{}{}
	}

	profilesContents, err := r.DS.GetMDMAndroidProfilesContents(ctx, slices.Collect(maps.Keys(profilesToLoad)))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load android profiles content")
	}

	// index the to-remove profiles by host so we can pass them to sendHostProfiles
	toRemoveByHostUUID := make(map[string][]*fleet.MDMAndroidProfilePayload)
	for _, prof := range hostsProfsToRemove {
		toRemoveByHostUUID[prof.HostUUID] = append(toRemoveByHostUUID[prof.HostUUID], prof)
	}

	var bulkHostProfs []*fleet.MDMAndroidProfilePayload
	for hostUUID, toInstallProfs := range profilesByHostUUID {
		toRemove := toRemoveByHostUUID[hostUUID]
		bulkProfs, err := r.sendHostProfiles(ctx, hostUUID, toInstallProfs, toRemove, profilesContents)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "send profiles for host %s", hostUUID)
		}
		bulkHostProfs = append(bulkHostProfs, bulkProfs...)
		delete(toRemoveByHostUUID, hostUUID)
	}

	// if there are hosts with only profiles to remove, process them too
	for hostUUID, toRemove := range toRemoveByHostUUID {
		bulkProfs, err := r.sendHostProfiles(ctx, hostUUID, nil, toRemove, nil)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "send profiles for host %s", hostUUID)
		}
		bulkHostProfs = append(bulkHostProfs, bulkProfs...)
	}

	if err := r.DS.BulkUpsertMDMAndroidHostProfiles(ctx, bulkHostProfs); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk upsert android host profiles")
	}
	return nil
}

func (r *profileReconciler) sendHostProfiles(
	ctx context.Context,
	hostUUID string,
	profilesToMerge []*fleet.MDMAndroidProfilePayload,
	profilesToRemove []*fleet.MDMAndroidProfilePayload,
	profilesContents map[string]json.RawMessage,
) ([]*fleet.MDMAndroidProfilePayload, error) {
	const maxRequestFailures = 3

	// We need a deterministic order to merge the profiles, and I opted to go by
	// name, alphabetically ascending, as it's simple, deterministic (names are
	// unique) and the ordering can be viewed by the user in the UI. We had also
	// discussed upload time of the profile but it may not be deterministic for
	// batch-set profiles (same timestamp when inserted in a transaction) and is
	// not readily visible in the UI.
	slices.SortFunc(profilesToMerge, func(a, b *fleet.MDMAndroidProfilePayload) int {
		return cmp.Compare(a.ProfileName, b.ProfileName)
	})

	// map of the bulk struct keyed by profile UUID
	bulkProfilesByUUID := make(map[string]*fleet.MDMAndroidProfilePayload, len(profilesToMerge)+len(profilesToRemove))

	// if every profile to install has > max failures, mark all as failed and done.
	setFailCount := initRequestFailCountForSetOfProfiles(profilesToMerge, profilesToRemove)
	if setFailCount >= maxRequestFailures {
		detail := `Couldn't apply profile. Google returned error. Please re-add profile to try again.`
		for _, prof := range profilesToMerge {
			bulkProfilesByUUID[prof.ProfileUUID] = &fleet.MDMAndroidProfilePayload{
				HostUUID:      hostUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeInstall,
				ProfileUUID:   prof.ProfileUUID,
				ProfileName:   prof.ProfileName,
				Detail:        detail,
			}
		}
		for _, prof := range profilesToRemove {
			bulkProfilesByUUID[prof.ProfileUUID] = &fleet.MDMAndroidProfilePayload{
				HostUUID:      hostUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
				ProfileUUID:   prof.ProfileUUID,
				ProfileName:   prof.ProfileName,
				Detail:        detail,
			}
		}
		return slices.Collect(maps.Values(bulkProfilesByUUID)), nil
	}

	// merge the profiles in order, keeping track of what profile overrides what
	// other one.
	settingFromProfile := make(map[string]string)   // setting name -> "winning" profile UUID
	overriddenSettings := make(map[string][]string) // profile UUID -> overridden setting names
	var finalJSON map[string]json.RawMessage
	for _, prof := range profilesToMerge {
		content, ok := profilesContents[prof.ProfileUUID]
		if !ok {
			// should never happen
			return nil, ctxerr.Errorf(ctx, "missing content for profile %s", prof.ProfileUUID)
		}

		var profJSON map[string]json.RawMessage
		if err := json.Unmarshal(content, &profJSON); err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "unmarshal profile %s content", prof.ProfileUUID)
		}

		if finalJSON == nil {
			finalJSON = profJSON
			for k := range profJSON {
				settingFromProfile[k] = prof.ProfileUUID
			}
		} else {
			for k, v := range profJSON {
				if _, alreadySet := finalJSON[k]; alreadySet {
					failedProfUUID := settingFromProfile[k]
					overriddenSettings[failedProfUUID] = append(overriddenSettings[failedProfUUID], k)
				}
				finalJSON[k] = v
				settingFromProfile[k] = prof.ProfileUUID
			}
		}

		status := fleet.MDMDeliveryPending
		bulkProfilesByUUID[prof.ProfileUUID] = &fleet.MDMAndroidProfilePayload{
			HostUUID:         hostUUID,
			Status:           &status,
			OperationType:    fleet.MDMOperationTypeInstall,
			ProfileUUID:      prof.ProfileUUID,
			ProfileName:      prof.ProfileName,
			RequestFailCount: setFailCount,
		}
	}

	// mark overridden profiles as failed
	for profUUID, overridden := range overriddenSettings {
		if len(overridden) > 0 {
			bulk := bulkProfilesByUUID[profUUID]
			bulk.Status = &fleet.MDMDeliveryFailed
			bulk.Detail = buildPolicyFieldsOverriddenErrorMessage(overridden)
		}
	}

	for _, prof := range profilesToRemove {
		status := fleet.MDMDeliveryPending
		bulkProfilesByUUID[prof.ProfileUUID] = &fleet.MDMAndroidProfilePayload{
			HostUUID:         hostUUID,
			Status:           &status,
			OperationType:    fleet.MDMOperationTypeRemove,
			ProfileUUID:      prof.ProfileUUID,
			ProfileName:      prof.ProfileName,
			RequestFailCount: setFailCount,
		}
	}

	// unmarshal the final JSON into the AMAPI policy struct
	var policy androidmanagement.Policy
	if finalJSON != nil {
		b, err := json.Marshal(finalJSON)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "marshal generic map of merged json")
		}
		if err := json.Unmarshal(b, &policy); err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "unmarshal merged json into policy struct")
		}
	}

	// for every policy (even empty), we want to enforce some settings
	applyFleetEnforcedSettings(&policy)

	// using the host uuid as policy id, so we don't need to track the id mapping
	// to the host.
	policyName := fmt.Sprintf("%s/policies/%s", r.Enterprise.Name(), hostUUID)
	policyReq, skip, err := r.patchPolicy(ctx, hostUUID, policyName, &policy, settingFromProfile)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "patch policy for host %s", hostUUID)
	}

	// set the policy request information on every profile that was part of it
	patchPolicyReqFailed := !skip && policyReq.StatusCode != http.StatusOK
	for _, prof := range bulkProfilesByUUID {
		prof.PolicyRequestUUID = &policyReq.RequestUUID
		if patchPolicyReqFailed {
			prof.RequestFailCount++
			prof.Status = nil // stays nil so it gets retried
		} else {
			prof.RequestFailCount = 0
			if policyReq.PolicyVersion.Valid {
				v := int(policyReq.PolicyVersion.V)
				prof.IncludedInPolicyVersion = &v
			}
		}
	}
	if patchPolicyReqFailed {
		return slices.Collect(maps.Values(bulkProfilesByUUID)), nil
	}

	// skip indicates that there was no change in the policy
	var androidHost *fleet.AndroidHost
	if !skip {
		// check if we need to patch the device too (if that policy name is not already
		// associated with the device)
		androidHost, err = r.DS.AndroidHostLiteByHostUUID(ctx, hostUUID)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "get android host by host UUID %s", hostUUID)
		}
		if androidHost.AppliedPolicyID != nil && *androidHost.AppliedPolicyID == hostUUID {
			skip = true
		}
	}

	if !skip {
		// we need to associate the device with that policy
		deviceName := fmt.Sprintf("%s/devices/%s", r.Enterprise.Name(), androidHost.DeviceID)
		device := &androidmanagement.Device{
			PolicyName: policyName,
			// State must be specified when updating a device, otherwise it fails with
			// "Illegal state transition from ACTIVE to DEVICE_STATE_UNSPECIFIED"
			//
			// > Note that when calling enterprises.devices.patch, ACTIVE and
			// > DISABLED are the only allowable values.

			// TODO(ap): should we send whatever the previous state was? If it was DISABLED,
			// we probably don't want to re-enable it by accident. Those are the only
			// 2 valid states when patching a device.
			State: "ACTIVE",
		}
		deviceReq, skip, err := r.patchDevice(ctx, hostUUID, deviceName, device)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "patch device for host %s", hostUUID)
		}

		// set the device request information on every profile that was part of it
		deviceReqFailed := !skip && deviceReq.StatusCode != http.StatusOK
		for _, prof := range bulkProfilesByUUID {
			prof.DeviceRequestUUID = &deviceReq.RequestUUID
			if deviceReqFailed {
				prof.RequestFailCount++
				prof.Status = nil // stays nil so it gets retried
			} else {
				prof.RequestFailCount = 0
			}
		}
	}

	return slices.Collect(maps.Values(bulkProfilesByUUID)), nil
}

func initRequestFailCountForSetOfProfiles(toInstall, toRemove []*fleet.MDMAndroidProfilePayload) int {
	// Use the smallest fail count as the starting point for the new profiles
	// (because it should be reset whenever the merged profile is different).
	count := -1
	for _, prof := range toInstall {
		if count == -1 || prof.RequestFailCount < count {
			count = prof.RequestFailCount
		}
	}
	for _, prof := range toRemove {
		if count == -1 || prof.RequestFailCount < count {
			count = prof.RequestFailCount
		}
	}
	if count == -1 {
		// should never happen, but just in case
		count = 0
	}
	return count
}

func buildPolicyFieldsOverriddenErrorMessage(overriddenFields []string) string {
	slices.Sort(overriddenFields)

	var sb strings.Builder
	for range len(overriddenFields) - 1 {
		sb.WriteString("%q, ")
	}
	if len(overriddenFields) > 1 {
		sb.WriteString("and ")
	}
	sb.WriteString("%q")
	if len(overriddenFields) > 1 {
		sb.WriteString(" aren't applied. They are overridden by other configuration profile.")
	} else {
		sb.WriteString(" isn't applied. It's overridden by other configuration profile.")
	}

	args := make([]any, len(overriddenFields))
	for i, s := range overriddenFields {
		args[i] = s
	}
	return fmt.Sprintf(sb.String(), args...)
}

func (r *profileReconciler) patchPolicy(ctx context.Context, policyID, policyName string,
	policy *androidmanagement.Policy, metadata map[string]string,
) (req *fleet.MDMAndroidPolicyRequest, skip bool, err error) {
	policyRequest, err := newAndroidPolicyRequest(policyID, policyName, policy, metadata)
	if err != nil {
		return nil, false, ctxerr.Wrapf(ctx, err, "prepare policy request %s", policyName)
	}

	applied, apiErr := r.Client.EnterprisesPoliciesPatch(ctx, policyName, policy)
	if apiErr != nil {
		var gerr *googleapi.Error
		if errors.As(apiErr, &gerr) {
			policyRequest.StatusCode = gerr.Code
		}
		policyRequest.ErrorDetails.V = apiErr.Error()
		policyRequest.ErrorDetails.Valid = true

		// Note that from my tests, the "not modified" error is not reliable, the
		// AMAPI happily returned 200 even if the policy was the same (as
		// confirmed by the same version number being returned), so we do check
		// for this error, but do not build critical logic on top of it.
		//
		// Tests do show that the version number is properly incremented when the
		// policy changes, though.
		if skip = androidmgmt.IsNotModifiedError(apiErr); skip {
			apiErr = nil
		}
	} else {
		policyRequest.StatusCode = http.StatusOK
		policyRequest.PolicyVersion.V = applied.Version
		policyRequest.PolicyVersion.Valid = true
	}

	if err := r.DS.NewAndroidPolicyRequest(ctx, policyRequest); err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "save android policy request")
	}
	return policyRequest, skip, nil
}

func newAndroidPolicyRequest(policyID, policyName string, policy *androidmanagement.Policy, metadata map[string]string) (*fleet.MDMAndroidPolicyRequest, error) {
	// save the payload with metadata about what setting comes from what profile
	m := fleet.AndroidPolicyRequestPayload{
		Policy: policy,
		Metadata: fleet.AndroidPolicyRequestPayloadMetadata{
			SettingsOrigin: metadata,
		},
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy to json: %w", err)
	}
	return &fleet.MDMAndroidPolicyRequest{
		RequestName: policyName,
		PolicyID:    policyID,
		Payload:     b,
	}, nil
}

func (r *profileReconciler) patchDevice(ctx context.Context, policyID, deviceName string,
	device *androidmanagement.Device,
) (req *fleet.MDMAndroidPolicyRequest, skip bool, apiErr error) {
	deviceRequest, err := newAndroidDeviceRequest(policyID, deviceName, device)
	if err != nil {
		return nil, false, ctxerr.Wrapf(ctx, err, "prepare device request %s", deviceName)
	}

	applied, apiErr := r.Client.EnterprisesDevicesPatch(ctx, deviceName, device)
	if apiErr != nil {
		var gerr *googleapi.Error
		if errors.As(apiErr, &gerr) {
			deviceRequest.StatusCode = gerr.Code
		}
		deviceRequest.ErrorDetails.V = apiErr.Error()
		deviceRequest.ErrorDetails.Valid = true

		if skip = androidmgmt.IsNotModifiedError(apiErr); skip {
			apiErr = nil
		}
	} else {
		deviceRequest.StatusCode = http.StatusOK
		deviceRequest.AppliedPolicyVersion.V = applied.AppliedPolicyVersion
		deviceRequest.AppliedPolicyVersion.Valid = true
	}

	if err := r.DS.NewAndroidPolicyRequest(ctx, deviceRequest); err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "save android device request")
	}
	return deviceRequest, skip, nil
}

func newAndroidDeviceRequest(policyID, deviceName string, device *androidmanagement.Device) (*fleet.MDMAndroidPolicyRequest, error) {
	b, err := json.Marshal(device)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal device to json: %w", err)
	}
	return &fleet.MDMAndroidPolicyRequest{
		RequestName: deviceName,
		PolicyID:    policyID,
		Payload:     b,
	}, nil
}

func applyFleetEnforcedSettings(policy *androidmanagement.Policy) {
	policy.StatusReportingSettings = &androidmanagement.StatusReportingSettings{
		DeviceSettingsEnabled:        true,
		MemoryInfoEnabled:            true,
		NetworkInfoEnabled:           true,
		DisplayInfoEnabled:           true,
		PowerManagementEventsEnabled: true,
		HardwareStatusEnabled:        true,
		SystemPropertiesEnabled:      true,
		SoftwareInfoEnabled:          true,
		CommonCriteriaModeEnabled:    true,
		ApplicationReportsEnabled:    true,
		ApplicationReportingSettings: nil, // only option is "includeRemovedApps", which I opted not to enable (we can diff apps to see removals)
	}
}

// reconcileCertificateTemplates processes certificate templates for Android in host batches.
func (r *profileReconciler) reconcileCertificateTemplates(ctx context.Context) error {
	const batchSize = 1000 // Process 1000 hosts at a time
	offset := 0

	for {
		// Get a batch of host UUIDs that have certificate templates
		hostUUIDs, err := r.DS.ListAndroidHostUUIDsWithCertificateTemplates(ctx, offset, batchSize)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list android host uuids with certificate templates")
		}

		if len(hostUUIDs) == 0 {
			break
		}

		// Get ALL certificate templates for this batch of hosts
		allTemplates, err := r.DS.ListCertificateTemplatesForHosts(ctx, hostUUIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list certificate templates for hosts")
		}

		// Process this batch of hosts with all their certificates
		if err := r.processCertificateTemplateBatch(ctx, allTemplates); err != nil {
			return err
		}

		if len(hostUUIDs) < batchSize {
			break
		}

		// next batch
		offset += batchSize
	}

	return nil
}

func (r *profileReconciler) processCertificateTemplateBatch(ctx context.Context, allTemplates []fleet.CertificateTemplateForHost) error {
	hostsWithNewCerts := make(map[string]struct{})
	newCertificates := make([]fleet.HostCertificateTemplate, 0)

	for i := range allTemplates {
		// Check if this is a new certificate (no existing record)
		// We should only get templates without existing records
		// But double-check just in case
		if allTemplates[i].FleetChallenge == nil {
			challenge, err := r.DS.NewChallenge(ctx)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "generate fleet challenge")
			}

			// Update the template with the challenge for later use
			allTemplates[i].FleetChallenge = &challenge
			hostsWithNewCerts[allTemplates[i].HostUUID] = struct{}{}

			// Track this as a new certificate to insert after successful delivery
			newCertificates = append(newCertificates, fleet.HostCertificateTemplate{
				HostUUID:              allTemplates[i].HostUUID,
				CertificateTemplateID: allTemplates[i].CertificateTemplateID,
				FleetChallenge:        challenge,
				Status:                fleet.MDMDeliveryPending,
			})
		}
	}

	// If no new certificates, we're done
	if len(newCertificates) == 0 {
		return nil
	}

	// Group ALL certificates by host UUID
	hostsNeedingUpdate := make(map[string][]fleet.CertificateTemplateForHost)
	for _, template := range allTemplates {
		if _, ok := hostsWithNewCerts[template.HostUUID]; ok {
			hostsNeedingUpdate[template.HostUUID] = append(hostsNeedingUpdate[template.HostUUID], template)
		}
	}

	hostConfigs := make(map[string]android.AgentManagedConfiguration)
	appConfig, err := r.DS.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get app config")
	}

	for hostUUID, certTemplates := range hostsNeedingUpdate {
		androidHost, err := r.DS.AndroidHostLiteByHostUUID(ctx, hostUUID)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "get android host %s", hostUUID)
		}

		enrollSecrets, err := r.DS.GetEnrollSecrets(ctx, androidHost.Host.TeamID)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "get enroll secrets for team %v", androidHost.Host.TeamID)
		}
		if len(enrollSecrets) == 0 {
			return ctxerr.Errorf(ctx, "no enroll secrets found for team %v", androidHost.Host.TeamID)
		}

		// Build certificate list with ALL certificates for this host
		agentCerts := make([]android.AgentCertificateTemplate, len(certTemplates))
		for i, ct := range certTemplates {
			// FleetChallenge should never be nil at this point since we only process hosts with new certs
			// But we'll dereference safely
			challenge := ""
			if ct.FleetChallenge != nil {
				challenge = *ct.FleetChallenge
			}
			agentCerts[i] = android.NewAgentCertificateTemplate(
				appConfig.ServerSettings.ServerURL,
				ct.CertificateTemplateID,
				hostUUID,
				challenge,
			)
		}

		hostConfigs[hostUUID] = android.AgentManagedConfiguration{
			ServerURL:            appConfig.ServerSettings.ServerURL,
			HostUUID:             hostUUID,
			EnrollSecret:         enrollSecrets[0].Secret,
			CertificateTemplates: agentCerts,
		}
	}

	// Update the Fleet Agent with certificate data
	svc := &Service{
		ds:               r.DS,
		androidAPIClient: r.Client,
	}
	if err := svc.AddFleetAgentToAndroidPolicy(ctx, r.Enterprise.Name(), hostConfigs); err != nil {
		return ctxerr.Wrap(ctx, err, "add fleet agent to android policy with certificates")
	}

	// For new certificates
	if err := r.DS.BulkInsertHostCertificateTemplates(ctx, newCertificates); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk insert host certificate templates")
	}

	return nil
}
