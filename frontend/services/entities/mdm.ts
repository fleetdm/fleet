/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import { DiskEncryptionStatus, MdmProfileStatus } from "interfaces/mdm";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export interface IEulaMetadataResponse {
  name: string;
  token: string;
  created_at: string;
}

export type ProfileStatusSummaryResponse = Record<MdmProfileStatus, number>;

export interface IDiskEncryptionStatusAggregate {
  macos: number;
  windows: number;
}

export type IDiskEncryptionSummaryResponse = Record<
  DiskEncryptionStatus,
  IDiskEncryptionStatusAggregate
>;

// This function combines the profile status summary and the disk encryption summary
// to generate the aggregate profile status summary. We are doing this as a temporary
// solution until we have the API that will return the aggregate profile status summary
// from one call.
// TODO: API INTEGRATION: remove when API is implemented that returns windows
// data in the aggregate profile status summary.
const generateCombinedProfileStatusSummary = (
  profileStatuses: ProfileStatusSummaryResponse,
  diskEncryptionSummary: IDiskEncryptionSummaryResponse
): ProfileStatusSummaryResponse => {
  const { verified, verifying, failed, pending } = profileStatuses;
  const {
    verified: verifiedDiskEncryption,
    verifying: verifyingDiskEncryption,
    failed: failedDiskEncryption,
    action_required: actionRequiredDiskEncryption,
    enforcing: enforcingDiskEncryption,
    removing_enforcement: removingEnforcementDiskEncryption,
  } = diskEncryptionSummary;

  return {
    verified: verified + verifiedDiskEncryption.windows,
    verifying: verifying + verifyingDiskEncryption.windows,
    failed: failed + failedDiskEncryption.windows,
    pending:
      pending +
      actionRequiredDiskEncryption.windows +
      enforcingDiskEncryption.windows +
      removingEnforcementDiskEncryption.windows,
  };
};

const mdmService = {
  downloadDeviceUserEnrollmentProfile: (token: string) => {
    const { DEVICE_USER_MDM_ENROLLMENT_PROFILE } = endpoints;
    return sendRequest("GET", DEVICE_USER_MDM_ENROLLMENT_PROFILE(token));
  },
  resetEncryptionKey: (token: string) => {
    const { DEVICE_USER_RESET_ENCRYPTION_KEY } = endpoints;
    return sendRequest("POST", DEVICE_USER_RESET_ENCRYPTION_KEY(token));
  },
  unenrollHostFromMdm: (hostId: number, timeout?: number) => {
    const { HOST_MDM_UNENROLL } = endpoints;
    return sendRequest(
      "PATCH",
      HOST_MDM_UNENROLL(hostId),
      undefined,
      undefined,
      timeout
    );
  },
  requestCSR: (email: string, organization: string) => {
    const { MDM_REQUEST_CSR } = endpoints;

    return sendRequest("POST", MDM_REQUEST_CSR, {
      email_address: email,
      organization,
    });
  },

  getProfiles: (teamId = APP_CONTEXT_NO_TEAM_ID) => {
    const { MDM_PROFILES_OLD: MDM_PROFILES } = endpoints;
    const path = `${MDM_PROFILES}?${buildQueryStringFromParams({
      team_id: teamId,
    })}`;

    return sendRequest("GET", path);
  },

  uploadProfile: (file: File, teamId?: number) => {
    const { MDM_PROFILES } = endpoints;

    const formData = new FormData();
    formData.append("profile", file);

    if (teamId) {
      formData.append("team_id", teamId.toString());
    }

    return sendRequest("POST", MDM_PROFILES, formData);
  },

  downloadProfile: (profileId: number) => {
    const { MDM_PROFILE } = endpoints;
    const path = `${MDM_PROFILE(profileId)}?${buildQueryStringFromParams({
      alt: "media",
    })}`;
    return sendRequest("GET", path);
  },

  deleteProfile: (profileId: number) => {
    const { MDM_PROFILE } = endpoints;
    return sendRequest("DELETE", MDM_PROFILE(profileId));
  },

  // TODO: API INTEGRATION: we need to rework this when we create API call that
  // will return the aggregate statuses for windows included in the response.
  // Currently to get windows data included we will need to make a separate call.
  // We will likely change this to go back to single "getProfileStatusSummary" API call.
  getAggregateProfileStatuses: async (
    teamId = APP_CONTEXT_NO_TEAM_ID,
    // TODO: WINDOWS FEATURE FLAG: remove when we windows feature is released.
    includeWindows: boolean
  ) => {
    // if we are not including windows we can just call the existing profile summary API
    if (!includeWindows) {
      return mdmService.getProfileStatusSummary(teamId);
    }

    // otherwise we have to make two calls and combine the results.
    return mdmService
      .getAggregateProfileStatusesWithWindows(teamId)
      .then((res) => generateCombinedProfileStatusSummary(...res));
  },

  getAggregateProfileStatusesWithWindows: async (teamId: number) => {
    return Promise.all([
      mdmService.getProfileStatusSummary(teamId),
      mdmService.getDiskEncryptionSummary(teamId),
    ]);
  },

  getProfileStatusSummary: (teamId = APP_CONTEXT_NO_TEAM_ID) => {
    const path = `${
      endpoints.MDM_PROFILES_AGGREGATE_STATUSES
    }?${buildQueryStringFromParams({ team_id: teamId })}`;
    return sendRequest("GET", path);
  },

  getDiskEncryptionSummary: (teamId?: number) => {
    let { MDM_DISK_ENCRYPTION_SUMMARY: path } = endpoints;

    if (teamId) {
      path = `${path}?${buildQueryStringFromParams({ team_id: teamId })}`;
    }
    return sendRequest("GET", path);
  },

  // TODO: API INTEGRATION: change when API is implemented that works for windows
  // disk encryption too.
  updateAppleMdmSettings: (enableDiskEncryption: boolean, teamId?: number) => {
    const {
      MDM_UPDATE_APPLE_SETTINGS: teamsEndpoint,
      CONFIG: noTeamsEndpoint,
    } = endpoints;
    if (teamId === 0) {
      return sendRequest("PATCH", noTeamsEndpoint, {
        mdm: {
          // TODO: API INTEGRATION: remove macos_settings when API change is merged in.
          macos_settings: { enable_disk_encryption: enableDiskEncryption },
          // enable_disk_encryption: enableDiskEncryption,
        },
      });
    }
    return sendRequest("PATCH", teamsEndpoint, {
      enable_disk_encryption: enableDiskEncryption,
      team_id: teamId,
    });
  },

  initiateMDMAppleSSO: () => {
    const { MDM_APPLE_SSO } = endpoints;
    return sendRequest("POST", MDM_APPLE_SSO, {});
  },

  getBootstrapPackageMetadata: (teamId: number) => {
    const { MDM_BOOTSTRAP_PACKAGE_METADATA } = endpoints;

    return sendRequest("GET", MDM_BOOTSTRAP_PACKAGE_METADATA(teamId));
  },

  uploadBootstrapPackage: (file: File, teamId?: number) => {
    const { MDM_BOOTSTRAP_PACKAGE } = endpoints;

    const formData = new FormData();
    formData.append("package", file);

    if (teamId) {
      formData.append("team_id", teamId.toString());
    }

    return sendRequest("POST", MDM_BOOTSTRAP_PACKAGE, formData);
  },

  deleteBootstrapPackage: (teamId: number) => {
    const { MDM_BOOTSTRAP_PACKAGE } = endpoints;
    return sendRequest("DELETE", `${MDM_BOOTSTRAP_PACKAGE}/${teamId}`);
  },

  getBootstrapPackageAggregate: (teamId?: number) => {
    let { MDM_BOOTSTRAP_PACKAGE_SUMMARY: path } = endpoints;

    if (teamId) {
      path = `${path}?${buildQueryStringFromParams({ team_id: teamId })}`;
    }

    return sendRequest("GET", path);
  },

  getEULAMetadata: () => {
    const { MDM_EULA_METADATA } = endpoints;
    return sendRequest("GET", MDM_EULA_METADATA);
  },

  uploadEULA: (file: File) => {
    const { MDM_EULA_UPLOAD } = endpoints;

    const formData = new FormData();
    formData.append("eula", file);

    return sendRequest("POST", MDM_EULA_UPLOAD, formData);
  },

  deleteEULA: (token: string) => {
    const { MDM_EULA } = endpoints;
    return sendRequest("DELETE", MDM_EULA(token));
  },

  downloadEULA: (token: string) => {
    const { MDM_EULA } = endpoints;
    return sendRequest("GET", MDM_EULA(token));
  },

  updateEndUserAuthentication: (teamId: number, isEnabled: boolean) => {
    const { MDM_SETUP } = endpoints;
    return sendRequest("PATCH", MDM_SETUP, {
      team_id: teamId,
      enable_end_user_authentication: isEnabled,
    });
  },
};

export default mdmService;
