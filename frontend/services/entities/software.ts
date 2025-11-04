import { AxiosProgressEvent } from "axios";

import sendRequest, { sendRequestWithProgress } from "services";
import endpoints from "utilities/endpoints";
import {
  ISoftwareResponse,
  ISoftwareCountResponse,
  ISoftwareVersion,
  ISoftwareTitle,
  ISoftwareTitleDetails,
  IFleetMaintainedApp,
  IFleetMaintainedAppDetails,
  ISoftwarePackage,
  SoftwareCategory,
} from "interfaces/software";
import {
  ApplePlatform,
  CommaSeparatedPlatformString,
} from "interfaces/platform";
import {
  buildQueryStringFromParams,
  convertParamsToSnakeCase,
  getPathWithQueryParams,
} from "utilities/url";
import { IPackageFormData } from "pages/SoftwarePage/components/forms/PackageForm/PackageForm";
import { IEditPackageFormData } from "pages/SoftwarePage/SoftwareTitleDetailsPage/EditSoftwareModal/EditSoftwareModal";
import { ISoftwareVppFormData } from "pages/SoftwarePage/components/forms/SoftwareVppForm/SoftwareVppForm";
import { IAddFleetMaintainedData } from "pages/SoftwarePage/SoftwareAddPage/SoftwareFleetMaintained/FleetMaintainedAppDetailsPage/FleetMaintainedAppDetailsPage";
import { listNamesFromSelectedLabels } from "components/TargetLabelSelector/TargetLabelSelector";
import { ISoftwareAndroidFormData } from "pages/SoftwarePage/components/forms/SoftwareAndroidForm/SoftwareAndroidForm";

export interface ISoftwareApiParams {
  page?: number;
  perPage?: number;
  orderKey?: string;
  orderDirection?: "asc" | "desc";
  query?: string;
  vulnerable?: boolean;
  max_cvss_score?: number;
  min_cvss_score?: number;
  exploit?: boolean;
  availableForInstall?: boolean;
  packagesOnly?: boolean;
  selfService?: boolean;
  teamId?: number;
}

export interface ISoftwareTitlesResponse {
  counts_updated_at: string | null;
  count: number;
  software_titles: ISoftwareTitle[];
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export interface ISoftwareVersionsResponse {
  counts_updated_at: string | null;
  count: number;
  software: ISoftwareVersion[];
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export interface ISoftwareTitleResponse {
  software_title: ISoftwareTitleDetails;
}

export interface ISoftwareVersionResponse {
  software: ISoftwareVersion;
}

export interface ISoftwareVersionsQueryKey extends ISoftwareApiParams {
  // used to trigger software refetches from sibling pages
  addedSoftwareToken: string | null;
  scope: "software-versions";
}

export interface ISoftwareTitlesQueryKey extends ISoftwareApiParams {
  // used to trigger software refetches from sibling pages
  addedSoftwareToken?: string | null;
  platform?: CommaSeparatedPlatformString;
  scope: "software-titles";
}

export interface ISoftwareQueryKey extends ISoftwareApiParams {
  scope: "software";
}

export interface ISoftwareCountQueryKey
  extends Pick<ISoftwareApiParams, "query" | "vulnerable" | "teamId"> {
  scope: "softwareCount";
}

export interface IGetSoftwareTitleQueryParams {
  softwareId: number;
  teamId?: number;
}

export interface IGetSoftwareTitleQueryKey
  extends IGetSoftwareTitleQueryParams {
  scope: "softwareById";
}

export interface IGetSoftwareVersionQueryParams {
  versionId: number;
  teamId?: number;
}

export interface IGetSoftwareVersionQueryKey
  extends IGetSoftwareVersionQueryParams {
  scope: "softwareVersion";
}

export interface ISoftwareInstallTokenResponse {
  token: string;
}

export interface ISoftwareFleetMaintainedAppsQueryParams {
  team_id: number;
  query?: string;
  order_key?: string;
  order_direction?: "asc" | "desc";
  page?: number;
  per_page?: number;
}

export interface ISoftwareFleetMaintainedAppsResponse {
  fleet_maintained_apps: IFleetMaintainedApp[];
  count: number;
  apps_updated_at: string | null;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export interface IFleetMaintainedAppResponse {
  fleet_maintained_app: IFleetMaintainedAppDetails;
}

interface IAddFleetMaintainedAppPostBody {
  team_id: number;
  fleet_maintained_app_id: number;
  pre_install_query?: string;
  install_script?: string;
  post_install_script?: string;
  uninstall_script?: string;
  self_service?: boolean;
  automatic_install?: boolean;
  labels_include_any?: string[];
  labels_exclude_any?: string[];
  categories: string[];
}

export interface IAddAppStoreAppPostBody {
  app_store_id: string;
  team_id: number;
  platform: ApplePlatform | "android";
  // True by default for android apps
  self_service?: boolean;
  // No automatic_install on add Android app
  automatic_install?: boolean;
  labels_include_any?: string[];
  labels_exclude_any?: string[];
  categories?: SoftwareCategory[];
}

// 4.77 Edit for Android app is not yet available
export interface IEditAppStoreAppPostBody {
  team_id: number;
  self_service?: boolean;
  // No automatic_install on edit VPP or android app
  labels_include_any?: string[];
  labels_exclude_any?: string[];
  categories?: SoftwareCategory[];
}

const ORDER_KEY = "name";
const ORDER_DIRECTION = "asc";

export const MAX_FILE_SIZE_MB = 3000;
export const MAX_FILE_SIZE_BYTES = MAX_FILE_SIZE_MB * 1024 * 1024;

const handleAndroidForm = (
  teamId: number,
  formData: ISoftwareAndroidFormData
) => {
  const { SOFTWARE_APP_STORE_APPS } = endpoints;

  const body: IAddAppStoreAppPostBody = {
    app_store_id: formData.applicationID,
    team_id: teamId,
    platform: formData.platform,
    self_service: formData.selfService,
    automatic_install: formData.automaticInstall,
  };

  if (formData.categories && formData.categories.length > 0) {
    body.categories = formData.categories as SoftwareCategory[];
  }

  if (formData.targetType === "Custom") {
    const selectedLabels = listNamesFromSelectedLabels(formData.labelTargets);
    if (formData.customTarget === "labelsIncludeAny") {
      body.labels_include_any = selectedLabels;
    } else {
      body.labels_exclude_any = selectedLabels;
    }
  }

  return sendRequest("POST", SOFTWARE_APP_STORE_APPS, body);
};

const handleVppAppForm = (teamId: number, formData: ISoftwareVppFormData) => {
  const { SOFTWARE_APP_STORE_APPS } = endpoints;

  if (!formData.selectedApp) {
    throw new Error("Selected app is required. This should not happen.");
  }

  const body: IAddAppStoreAppPostBody = {
    app_store_id: formData.selectedApp.app_store_id,
    team_id: teamId,
    platform: formData.selectedApp?.platform, // Nested platform
    self_service: formData.selfService,
    automatic_install: formData.automaticInstall,
  };

  if (formData.categories && formData.categories.length > 0) {
    body.categories = formData.categories as SoftwareCategory[];
  }

  if (formData.targetType === "Custom") {
    const selectedLabels = listNamesFromSelectedLabels(formData.labelTargets);
    if (formData.customTarget === "labelsIncludeAny") {
      body.labels_include_any = selectedLabels;
    } else {
      body.labels_exclude_any = selectedLabels;
    }
  }

  return sendRequest("POST", SOFTWARE_APP_STORE_APPS, body);
};

export default {
  load: async ({
    page,
    perPage,
    orderKey = ORDER_KEY,
    orderDirection: orderDir = ORDER_DIRECTION,
    query,
    vulnerable,
    // availableForInstall, // TODO: Is this supported for the versions endpoint?
    teamId,
  }: Omit<
    ISoftwareApiParams,
    "availableForInstall" | "selfService"
  >): Promise<ISoftwareResponse> => {
    const { SOFTWARE } = endpoints;
    const queryParams = {
      page,
      perPage,
      orderKey,
      orderDirection: orderDir,
      teamId,
      query,
      vulnerable,
      // availableForInstall,
    };

    const snakeCaseParams = convertParamsToSnakeCase(queryParams);
    const queryString = buildQueryStringFromParams(snakeCaseParams);
    const path = `${SOFTWARE}?${queryString}`;

    try {
      return sendRequest("GET", path);
    } catch (error) {
      throw error;
    }
  },

  getCount: async ({
    query,
    teamId,
    vulnerable,
  }: Pick<
    ISoftwareApiParams,
    "query" | "teamId" | "vulnerable"
  >): Promise<ISoftwareCountResponse> => {
    const { SOFTWARE } = endpoints;
    const path = `${SOFTWARE}/count`;
    const queryParams = {
      query,
      teamId,
      vulnerable,
    };
    const snakeCaseParams = convertParamsToSnakeCase(queryParams);
    const queryString = buildQueryStringFromParams(snakeCaseParams);

    return sendRequest("GET", path.concat(`?${queryString}`));
  },

  getSoftwareTitles: (
    params: ISoftwareApiParams
  ): Promise<ISoftwareTitlesResponse> => {
    const { SOFTWARE_TITLES } = endpoints;
    const snakeCaseParams = convertParamsToSnakeCase(params);
    const queryString = buildQueryStringFromParams(snakeCaseParams);
    const path = `${SOFTWARE_TITLES}?${queryString}`;
    return sendRequest("GET", path);
  },

  getSoftwareTitle: ({
    softwareId,
    teamId,
  }: IGetSoftwareTitleQueryParams): Promise<ISoftwareTitleResponse> => {
    const endpoint = endpoints.SOFTWARE_TITLE(softwareId);
    const queryString = buildQueryStringFromParams({ team_id: teamId });
    const path =
      typeof teamId === "undefined" ? endpoint : `${endpoint}?${queryString}`;
    return sendRequest("GET", path);
  },

  getSoftwareVersions: (params: ISoftwareApiParams) => {
    const { SOFTWARE_VERSIONS } = endpoints;
    const snakeCaseParams = convertParamsToSnakeCase(params);
    const queryString = buildQueryStringFromParams(snakeCaseParams);
    const path = `${SOFTWARE_VERSIONS}?${queryString}`;
    return sendRequest("GET", path);
  },

  getSoftwareVersion: ({
    versionId,
    teamId,
  }: IGetSoftwareVersionQueryParams) => {
    const endpoint = endpoints.SOFTWARE_VERSION(versionId);
    const queryString = buildQueryStringFromParams({ team_id: teamId });
    const path =
      typeof teamId === "undefined" ? endpoint : `${endpoint}?${queryString}`;

    return sendRequest("GET", path);
  },

  addSoftwarePackage: ({
    data,
    teamId,
    timeout,
    onUploadProgress,
    signal,
  }: {
    data: IPackageFormData;
    teamId?: number;
    timeout?: number;
    onUploadProgress?: (progressEvent: AxiosProgressEvent) => void;
    signal?: AbortSignal;
  }) => {
    const { SOFTWARE_PACKAGE_ADD } = endpoints;

    if (!data.software) {
      throw new Error("Software package is required");
    }

    const formData = new FormData();
    formData.append("software", data.software);
    formData.append("self_service", data.selfService.toString());
    data.installScript && formData.append("install_script", data.installScript);
    data.uninstallScript &&
      formData.append("uninstall_script", data.uninstallScript);
    data.preInstallQuery &&
      formData.append("pre_install_query", data.preInstallQuery);
    data.postInstallScript &&
      formData.append("post_install_script", data.postInstallScript);
    data.automaticInstall &&
      formData.append("automatic_install", data.automaticInstall.toString());
    teamId && formData.append("team_id", teamId.toString());
    if (data.categories) {
      data.categories.forEach((category) => {
        formData.append("categories", category);
      });
    }

    if (data.targetType === "Custom") {
      const selectedLabels = listNamesFromSelectedLabels(data.labelTargets);
      let labelKey = "";
      if (data.customTarget === "labelsIncludeAny") {
        labelKey = "labels_include_any";
      } else {
        labelKey = "labels_exclude_any";
      }
      selectedLabels?.forEach((label) => {
        formData.append(labelKey, label);
      });
    }

    return sendRequestWithProgress({
      method: "POST",
      path: SOFTWARE_PACKAGE_ADD,
      data: formData,
      timeout,
      skipParseError: true,
      onUploadProgress,
      signal,
    });
  },

  editSoftwarePackage: ({
    data,
    orignalPackage,
    softwareId,
    teamId,
    timeout,
    onUploadProgress,
    signal,
  }: {
    data: IEditPackageFormData;
    orignalPackage: ISoftwarePackage;
    softwareId: number;
    teamId: number;
    timeout?: number;
    onUploadProgress?: (progressEvent: AxiosProgressEvent) => void;
    signal?: AbortSignal;
  }) => {
    const { EDIT_SOFTWARE_PACKAGE } = endpoints;

    const formData = new FormData();
    formData.append("team_id", teamId.toString());
    data.software && formData.append("software", data.software);
    formData.append("self_service", data.selfService.toString());
    formData.append("install_script", data.installScript);
    formData.append("pre_install_query", data.preInstallQuery || "");
    formData.append("post_install_script", data.postInstallScript || "");
    formData.append("uninstall_script", data.uninstallScript || "");
    if (data.categories) {
      data.categories.forEach((category) => {
        formData.append("categories", category);
      });
    }

    // clear out labels if targetType is "All hosts"
    if (data.targetType === "All hosts") {
      if (orignalPackage.labels_include_any) {
        formData.append("labels_include_any", "");
      } else {
        formData.append("labels_exclude_any", "");
      }
    }

    // add custom labels if targetType is "Custom"
    if (data.targetType === "Custom") {
      const selectedLabels = listNamesFromSelectedLabels(data.labelTargets);
      let labelKey = "";
      if (data.customTarget === "labelsIncludeAny") {
        labelKey = "labels_include_any";
      } else {
        labelKey = "labels_exclude_any";
      }
      selectedLabels?.forEach((label) => {
        formData.append(labelKey, label);
      });
    }

    return sendRequestWithProgress({
      method: "PATCH",
      path: EDIT_SOFTWARE_PACKAGE(softwareId),
      data: formData,
      timeout,
      skipParseError: true,
      onUploadProgress,
      signal,
    });
  },

  addAppStoreApp: (
    teamId: number,
    formData: ISoftwareVppFormData | ISoftwareAndroidFormData
  ) => {
    if ("platform" in formData) {
      // Android form data
      return handleAndroidForm(teamId, formData as ISoftwareAndroidFormData);
    }

    // Apple VPP form data
    return handleVppAppForm(teamId, formData as ISoftwareVppFormData);
  },

  editAppStoreApp: (
    softwareId: number,
    teamId: number,
    formData: ISoftwareVppFormData | ISoftwareAndroidFormData
  ) => {
    const { EDIT_SOFTWARE_APP_STORE_APP } = endpoints;

    const body: IEditAppStoreAppPostBody = {
      self_service: formData.selfService,
      team_id: teamId,
    };

    // Add categories if present
    if (formData.categories && formData.categories.length > 0) {
      body.categories = formData.categories as SoftwareCategory[];
    }

    if (formData.targetType === "Custom") {
      const selectedLabels = listNamesFromSelectedLabels(formData.labelTargets);
      if (formData.customTarget === "labelsIncludeAny") {
        body.labels_include_any = selectedLabels;
      } else {
        body.labels_exclude_any = selectedLabels;
      }
    }

    return sendRequest("PATCH", EDIT_SOFTWARE_APP_STORE_APP(softwareId), body);
  },

  getSoftwareIcon: (softwareId: number, teamId: number) => {
    const { SOFTWARE_ICON } = endpoints;
    const path = getPathWithQueryParams(SOFTWARE_ICON(softwareId), {
      team_id: teamId,
    });
    return sendRequest(
      "GET",
      path,
      undefined,
      "blob",
      undefined,
      undefined,
      true
    ); // returnRaw is true to get headers
  },

  // This API call is for both:
  // "/api/v1/fleet/software/titles/{softwareId}/icon?team_id={teamId}"
  // "/api/v1/fleet/device/{deviceToken}/software/titles/{softwareId}/icon"
  getSoftwareIconFromApiUrl: (apiUrl: string) => {
    // sendRequest prepends "/api" to the path, so we need to remove it
    // if it's already included in the apiUrl param
    const result = apiUrl.replace(/^\/api/, "");

    return sendRequest("GET", result, undefined, "blob");
  },

  deleteSoftwareIcon: (softwareId: number, teamId: number) => {
    const { SOFTWARE_ICON } = endpoints;
    const path = getPathWithQueryParams(SOFTWARE_ICON(softwareId), {
      team_id: teamId,
    });
    return sendRequest("DELETE", path);
  },

  editSoftwareIcon: (
    softwareId: number,
    teamId: number,
    fileObject: { icon: File }
  ) => {
    const { SOFTWARE_ICON } = endpoints;
    const path = getPathWithQueryParams(SOFTWARE_ICON(softwareId), {
      team_id: teamId,
    });

    const formData = new FormData();
    formData.append("icon", fileObject.icon);

    return sendRequest("PUT", path, formData);
  },

  // Endpoint for deleting packages or VPP
  deleteSoftwareInstaller: (softwareId: number, teamId: number) => {
    const { SOFTWARE_AVAILABLE_FOR_INSTALL } = endpoints;
    const path = `${SOFTWARE_AVAILABLE_FOR_INSTALL(
      softwareId
    )}?team_id=${teamId}`;
    return sendRequest("DELETE", path);
  },

  getSoftwarePackageToken: (
    softwareTitleId: number,
    teamId: number
  ): Promise<ISoftwareInstallTokenResponse> => {
    const path = `${endpoints.SOFTWARE_PACKAGE_TOKEN(
      softwareTitleId
    )}?${buildQueryStringFromParams({ alt: "media", team_id: teamId })}`;

    return sendRequest("POST", path);
  },

  getSoftwareInstallResult: (installUuid: string) => {
    const { SOFTWARE_INSTALL_RESULTS } = endpoints;
    const path = SOFTWARE_INSTALL_RESULTS(installUuid);
    return sendRequest("GET", path);
  },

  getFleetMaintainedApps: (
    params: ISoftwareFleetMaintainedAppsQueryParams
  ): Promise<ISoftwareFleetMaintainedAppsResponse> => {
    const { SOFTWARE_FLEET_MAINTAINED_APPS } = endpoints;
    const queryStr = buildQueryStringFromParams(params);
    const path = `${SOFTWARE_FLEET_MAINTAINED_APPS}?${queryStr}`;
    return sendRequest("GET", path);
  },

  getFleetMaintainedApp: (
    id: number,
    teamId?: string
  ): Promise<IFleetMaintainedAppResponse> => {
    const { SOFTWARE_FLEET_MAINTAINED_APP } = endpoints;
    const path = getPathWithQueryParams(SOFTWARE_FLEET_MAINTAINED_APP(id), {
      team_id: teamId,
    });
    return sendRequest("GET", path);
  },

  addFleetMaintainedApp: (
    teamId: number,
    formData: IAddFleetMaintainedData
  ) => {
    const { SOFTWARE_FLEET_MAINTAINED_APPS } = endpoints;

    const body: IAddFleetMaintainedAppPostBody = {
      team_id: teamId,
      fleet_maintained_app_id: formData.appId,
      pre_install_query: formData.preInstallQuery,
      install_script: formData.installScript,
      post_install_script: formData.postInstallScript,
      uninstall_script: formData.uninstallScript,
      self_service: formData.selfService,
      automatic_install: formData.automaticInstall,
      categories: formData.categories,
    };

    if (formData.targetType === "Custom") {
      const selectedLabels = listNamesFromSelectedLabels(formData.labelTargets);
      if (formData.customTarget === "labelsIncludeAny") {
        body.labels_include_any = selectedLabels;
      } else {
        body.labels_exclude_any = selectedLabels;
      }
    }

    return sendRequest("POST", SOFTWARE_FLEET_MAINTAINED_APPS, body);
  },
};
