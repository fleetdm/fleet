import { AxiosResponse } from "axios";

import sendRequest from "services";
import endpoints from "utilities/endpoints";
import {
  ISoftwareResponse,
  ISoftwareCountResponse,
  ISoftwareVersion,
  ISoftwareTitle,
  ISoftwareTitleDetails,
} from "interfaces/software";
import {
  buildQueryStringFromParams,
  convertParamsToSnakeCase,
} from "utilities/url";
import { IAddPackageFormData } from "pages/SoftwarePage/components/AddPackageForm/AddPackageForm";

export interface ISoftwareApiParams {
  page?: number;
  perPage?: number;
  orderKey?: string;
  orderDirection?: "asc" | "desc";
  query?: string;
  vulnerable?: boolean;
  availableForInstall?: boolean;
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

const ORDER_KEY = "name";
const ORDER_DIRECTION = "asc";

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

  addSoftwarePackage: (
    data: IAddPackageFormData,
    teamId?: number,
    timeout?: number
  ) => {
    const { SOFTWARE_PACKAGE_ADD } = endpoints;

    if (!data.software) {
      throw new Error("Software package is required");
    }

    const formData = new FormData();
    formData.append("software", data.software);
    formData.append("self_service", data.selfService.toString());
    formData.append("install_type", data.installType);
    data.installScript && formData.append("install_script", data.installScript);
    data.preInstallCondition &&
      formData.append("pre_install_query", data.preInstallCondition);
    data.postInstallScript &&
      formData.append("post_install_script", data.postInstallScript);

    if (data.useCustomTargets) {
      // handles when a label has been selected then de-selected
      const labelNameAcc: string[] = [];
      const selectedLabelNames = Object.entries(data.selectedLabels).reduce(
        (acc, [name, isSelected]) => {
          if (isSelected) {
            return [...acc, name];
          }
          return acc;
        },
        labelNameAcc
      );

      if (selectedLabelNames.length > 0) {
        const labelFieldName = `labels_${
          data.includeAnyLabels ? "include" : "exclude"
        }_any`;
        selectedLabelNames.forEach((labelName) => {
          formData.append(labelFieldName, labelName);
        });
      }
    }

    teamId && formData.append("team_id", teamId.toString());

    return sendRequest(
      "POST",
      SOFTWARE_PACKAGE_ADD,
      formData,
      undefined,
      timeout,
      true
    );
  },

  deleteSoftwarePackage: (softwareId: number, teamId: number) => {
    const { SOFTWARE_AVAILABLE_FOR_INSTALL } = endpoints;
    const path = `${SOFTWARE_AVAILABLE_FOR_INSTALL(
      softwareId
    )}?team_id=${teamId}`;
    return sendRequest("DELETE", path);
  },

  downloadSoftwarePackage: (
    softwareTitleId: number,
    teamId: number
  ): Promise<AxiosResponse> => {
    const path = `${endpoints.SOFTWARE_PACKAGE(
      softwareTitleId
    )}?${buildQueryStringFromParams({ alt: "media", team_id: teamId })}`;

    return sendRequest(
      "GET",
      path,
      undefined,
      "blob",
      undefined,
      undefined,
      true // return raw response
    );
  },

  getSoftwareInstallResult: (installUuid: string) => {
    const { SOFTWARE_INSTALL_RESULTS } = endpoints;
    const path = SOFTWARE_INSTALL_RESULTS(installUuid);
    return sendRequest("GET", path);
  },
};
