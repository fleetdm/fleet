/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import { DiskEncryptionStatus } from "interfaces/mdm";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export interface IEulaMetadataResponse {
  name: string;
  token: string;
  created_at: string;
}

export interface IDiskEncryptionStatusAggregate {
  macos: string;
  windows: string;
}

export type IDiskEncryptionSummaryResponse = Record<
  DiskEncryptionStatus,
  IDiskEncryptionStatusAggregate
>;

export default {
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
    const path = `${endpoints.MDM_PROFILES}?${buildQueryStringFromParams({
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
    return sendRequest("GET", MDM_PROFILE(profileId));
  },

  deleteProfile: (profileId: number) => {
    const { MDM_PROFILE } = endpoints;
    return sendRequest("DELETE", MDM_PROFILE(profileId));
  },

  getAggregateProfileStatuses: (teamId = APP_CONTEXT_NO_TEAM_ID) => {
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

    // TODO: change when API is implemented
    return new Promise<IDiskEncryptionSummaryResponse>((resolve) => {
      resolve({
        verified: { macos: "0", windows: "5" },
        verifying: { macos: "1", windows: "4" },
        action_required: { macos: "2", windows: "3" },
        enforcing: { macos: "3", windows: "2" },
        failed: { macos: "4", windows: "1" },
        removing_enforcement: { macos: "5", windows: "0" },
      });
    });
    // return sendRequest("GET", path);
  },

  updateAppleMdmSettings: (enableDiskEncryption: boolean, teamId?: number) => {
    const {
      MDM_UPDATE_APPLE_SETTINGS: teamsEndpoint,
      CONFIG: noTeamsEndpoint,
    } = endpoints;
    if (teamId === 0) {
      return sendRequest("PATCH", noTeamsEndpoint, {
        mdm: {
          macos_settings: { enable_disk_encryption: enableDiskEncryption },
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
