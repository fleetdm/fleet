import {
  IHostMdmProfile,
  IMdmCommandResult,
  IMdmProfile,
  MdmProfileStatus,
} from "interfaces/mdm";
import { API_NO_TEAM_ID } from "interfaces/team";
import { ISoftwareTitle } from "interfaces/software";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

import { ISoftwareTitlesResponse } from "./software";

export interface IEulaMetadataResponse {
  name: string;
  token: string;
  created_at: string;
}

export type ProfileStatusSummaryResponse = Record<MdmProfileStatus, number>;

export interface IGetProfilesApiParams {
  page?: number;
  per_page?: number;
  team_id?: number;
}

export interface IMdmProfilesResponse {
  profiles: IMdmProfile[] | null;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export interface IUploadProfileApiParams {
  file: File;
  teamId?: number;
  labelsIncludeAll?: string[];
  labelsIncludeAny?: string[];
  labelsExcludeAny?: string[];
}

export const isDDMProfile = (profile: IMdmProfile | IHostMdmProfile) => {
  return profile.profile_uuid.startsWith("d");
};

interface IUpdateSetupExperienceBody {
  team_id?: number;
  enable_release_device_manually: boolean;
}

export interface IAppleSetupEnrollmentProfileResponse {
  team_id: number | null;
  name: string;
  uploaded_at: string;
  // enrollment profile is an object with keys found here https://developer.apple.com/documentation/devicemanagement/profile.
  enrollment_profile: Record<string, unknown>;
}

export interface IMDMSSOParams {
  dep_device_info: string;
}

export interface IMDMAppleEnrollmentProfileParams {
  token: string;
  ref?: string;
  dep_device_info?: string;
}

export interface IGetMdmCommandResultsResponse {
  results: IMdmCommandResult[];
}

export interface IGetSetupExperienceScriptResponse {
  id: number;
  team_id: number | null; // The API return null for no team in this case.
  name: string;
  created_at: string;
  updated_at: string;
}

interface IGetSetupExperienceSoftwareParams {
  team_id: number;
  per_page: number;
}

export type IGetSetupExperienceSoftwareResponse = ISoftwareTitlesResponse & {
  software_titles: ISoftwareTitle[] | null;
};

const mdmService = {
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
  requestCSR: () => {
    const { MDM_REQUEST_CSR } = endpoints;

    return sendRequest("GET", MDM_REQUEST_CSR);
  },

  getProfiles: (
    params: IGetProfilesApiParams
  ): Promise<IMdmProfilesResponse> => {
    const { MDM_PROFILES } = endpoints;
    const path = `${MDM_PROFILES}?${buildQueryStringFromParams({
      ...params,
    })}`;

    return sendRequest("GET", path);
  },

  uploadProfile: ({
    file,
    teamId,
    labelsIncludeAll,
    labelsIncludeAny,
    labelsExcludeAny,
  }: IUploadProfileApiParams) => {
    const { MDM_PROFILES } = endpoints;

    const formData = new FormData();
    formData.append("profile", file);

    if (teamId) {
      formData.append("team_id", teamId.toString());
    }

    if (labelsIncludeAll || labelsIncludeAny || labelsExcludeAny) {
      const labels = labelsIncludeAll || labelsIncludeAny || labelsExcludeAny;

      let labelKey = "";
      if (labelsIncludeAll) {
        labelKey = "labels_include_all";
      } else if (labelsIncludeAny) {
        labelKey = "labels_include_any";
      } else {
        labelKey = "labels_exclude_any";
      }

      labels?.forEach((label) => {
        formData.append(labelKey, label);
      });
    }

    return sendRequest("POST", MDM_PROFILES, formData);
  },

  downloadProfile: (profileId: string) => {
    const { MDM_PROFILE } = endpoints;
    const path = `${MDM_PROFILE(profileId)}?${buildQueryStringFromParams({
      alt: "media",
    })}`;
    return sendRequest("GET", path);
  },

  deleteProfile: (profileId: string) => {
    const { MDM_PROFILE } = endpoints;
    return sendRequest("DELETE", MDM_PROFILE(profileId));
  },

  getProfilesStatusSummary: (teamId: number) => {
    let { MDM_PROFILES_STATUS_SUMMARY: path } = endpoints;

    if (teamId) {
      path = `${path}?${buildQueryStringFromParams({ team_id: teamId })}`;
    }

    return sendRequest("GET", path);
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

  updateReleaseDeviceSetting: (teamId: number, isEnabled: boolean) => {
    const { MDM_SETUP_EXPERIENCE } = endpoints;

    const body: IUpdateSetupExperienceBody = {
      enable_release_device_manually: isEnabled,
    };

    if (teamId !== API_NO_TEAM_ID) {
      body.team_id = teamId;
    }

    return sendRequest("PATCH", MDM_SETUP_EXPERIENCE, body);
  },

  getSetupEnrollmentProfile: (teamId?: number) => {
    const { MDM_APPLE_SETUP_ENROLLMENT_PROFILE } = endpoints;
    if (!teamId || teamId === API_NO_TEAM_ID) {
      return sendRequest("GET", MDM_APPLE_SETUP_ENROLLMENT_PROFILE);
    }

    const path = `${MDM_APPLE_SETUP_ENROLLMENT_PROFILE}?${buildQueryStringFromParams(
      { team_id: teamId }
    )}`;
    return sendRequest("GET", path);
  },

  uploadSetupEnrollmentProfile: (file: File, teamId: number) => {
    const { MDM_APPLE_SETUP_ENROLLMENT_PROFILE } = endpoints;

    const reader = new FileReader();
    reader.readAsText(file);

    return new Promise((resolve, reject) => {
      reader.addEventListener("load", () => {
        try {
          const body: Record<string, unknown> = {
            name: file.name,
            enrollment_profile: JSON.parse(reader.result as string),
          };
          if (teamId !== API_NO_TEAM_ID) {
            body.team_id = teamId;
          }
          resolve(
            sendRequest("POST", MDM_APPLE_SETUP_ENROLLMENT_PROFILE, body)
          );
        } catch {
          // catches invalid JSON
          reject("Couldn't upload. The file should include valid JSON.");
        }
      });
    });
  },

  deleteSetupEnrollmentProfile: (teamId: number) => {
    const { MDM_APPLE_SETUP_ENROLLMENT_PROFILE } = endpoints;
    if (teamId === API_NO_TEAM_ID) {
      return sendRequest("DELETE", MDM_APPLE_SETUP_ENROLLMENT_PROFILE);
    }

    const path = `${MDM_APPLE_SETUP_ENROLLMENT_PROFILE}?${buildQueryStringFromParams(
      { team_id: teamId }
    )}`;
    return sendRequest("DELETE", path);
  },

  getCommandResults: (
    command_uuid: string
  ): Promise<IGetMdmCommandResultsResponse> => {
    const { COMMANDS_RESULTS: MDM_COMMANDS_RESULTS } = endpoints;
    const url = `${MDM_COMMANDS_RESULTS}?command_uuid=${command_uuid}`;
    return sendRequest("GET", url);
  },

  downloadManualEnrollmentProfile: (token: string) => {
    const { DEVICE_USER_MDM_ENROLLMENT_PROFILE } = endpoints;
    return sendRequest(
      "GET",
      DEVICE_USER_MDM_ENROLLMENT_PROFILE(token),
      undefined,
      "blob"
    );
  },

  getSetupExperienceSoftware: (
    params: IGetSetupExperienceSoftwareParams
  ): Promise<IGetSetupExperienceSoftwareResponse> => {
    const { MDM_SETUP_EXPERIENCE_SOFTWARE } = endpoints;

    const path = `${MDM_SETUP_EXPERIENCE_SOFTWARE}?${buildQueryStringFromParams(
      {
        ...params,
      }
    )}`;

    return sendRequest("GET", path);
  },

  updateSetupExperienceSoftware: (
    teamId: number,
    softwareTitlesIds: number[]
  ) => {
    const { MDM_SETUP_EXPERIENCE_SOFTWARE } = endpoints;

    const path = `${MDM_SETUP_EXPERIENCE_SOFTWARE}?${buildQueryStringFromParams(
      {
        team_id: teamId,
      }
    )}`;

    return sendRequest("PUT", path, {
      team_id: teamId,
      software_title_ids: softwareTitlesIds,
    });
  },

  getSetupExperienceScript: (
    teamId: number
  ): Promise<IGetSetupExperienceScriptResponse> => {
    const { MDM_SETUP_EXPERIENCE_SCRIPT } = endpoints;

    let path = MDM_SETUP_EXPERIENCE_SCRIPT;
    if (teamId) {
      path += `?${buildQueryStringFromParams({ team_id: teamId })}`;
    }

    return sendRequest("GET", path);
  },

  downloadSetupExperienceScript: (teamId: number) => {
    const { MDM_SETUP_EXPERIENCE_SCRIPT } = endpoints;

    let path = MDM_SETUP_EXPERIENCE_SCRIPT;
    path += `?${buildQueryStringFromParams({ team_id: teamId, alt: "media" })}`;

    return sendRequest("GET", path);
  },

  uploadSetupExperienceScript: (file: File, teamId: number) => {
    const { MDM_SETUP_EXPERIENCE_SCRIPT } = endpoints;

    const formData = new FormData();
    formData.append("script", file);

    if (teamId) {
      formData.append("team_id", teamId.toString());
    }

    return sendRequest("POST", MDM_SETUP_EXPERIENCE_SCRIPT, formData);
  },

  deleteSetupExperienceScript: (teamId: number) => {
    const { MDM_SETUP_EXPERIENCE_SCRIPT } = endpoints;

    const path = `${MDM_SETUP_EXPERIENCE_SCRIPT}?${buildQueryStringFromParams({
      team_id: teamId,
    })}`;

    return sendRequest("DELETE", path);
  },
};

export default mdmService;
