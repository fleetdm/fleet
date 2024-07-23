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
import { IAddSoftwareFormData } from "pages/SoftwarePage/components/AddPackageForm/AddSoftwareForm";

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

// `GET /api/v1/fleet/software/titles`
export interface ISoftwareTitlesResponse {
  counts_updated_at: string | null;
  count: number;
  software_titles: ISoftwareTitle[];
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

// {
//   "counts_updated_at": "2022-01-01 12:32:00",
//   "count": 1,
//   "software_titles": [
//     {
//       "id": 12,
//       "name": "Firefox.app",
//       "software_package": {
//         "name": "FirefoxInsall.pkg",
//         "version": "125.6",
//         "self_service": true
//       },
//       "app_store_app": null,
//       "versions_count": 3,
//       "source": “ipados_apps", // | "ios_apps" | "apps"
//       "browser": "",
//       "hosts_count": 48,
//       "versions": [
//         {
//           "id": 123,
//           "version": "1.12",
//           "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"]
//         },
//         {
//           "id": 124,
//           "version": "3.4",
//           "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"]
//         },
//         {
//           "id": 12
//           "version": "1.13",
//           "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"]
//         }
//       ]
//     },
//   ],
//   meta: {
//     has_next_results: false,
//     has_previous_results: false,
// };
// }

// `GET /api/v1/fleet/software/versions/12`
export interface ISoftwareVersionsResponse {
  counts_updated_at: string | null;
  count: number;
  software: ISoftwareVersion[];
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}
// {
//   "software": {
//     "id": 425224,
//     "name": "Firefox.app",
//     "version": "117.0",
//     "bundle_identifier": "org.mozilla.firefox",
//     "source": "ipados_apps", # or `ios_apps`
//     "browser": "",
//     "generated_cpe": "cpe:2.3:a:mozilla:firefox:117.0:*:*:*:*:macos:*:*",
//     "vulnerabilities": [
//       {
//         "cve": "CVE-2023-4863",
//         "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2023-4863",
//         "created_at": "2024-07-01T00:15:00Z",
//         "cvss_score": 8.8, // Available in Fleet Premium
//         "epss_probability": 0.4101, // Available in Fleet Premium
//         "cisa_known_exploit": true, // Available in Fleet Premium
//         "cve_published": "2023-09-12T15:15:00Z", // Available in Fleet Premium
//         "resolved_in_version": "" // Available in Fleet Premium
//       },
//       {
//         "cve": "CVE-2023-5169",
//         "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2023-5169",
//         "created_at": "2024-07-01T00:15:00Z",
//         "cvss_score": 6.5, // Available in Fleet Premium
//         "epss_probability": 0.00073, // Available in Fleet Premium
//         "cisa_known_exploit": false, // Available in Fleet Premium
//         "cve_published": "2023-09-27T15:19:00Z", // Available in Fleet Premium
//         "resolved_in_version": "118" // Available in Fleet Premium
//       }
//     ]
//   }
// }

// `GET /api/v1/fleet/software/titles/:id`

export interface ISoftwareTitleResponse {
  software_title: ISoftwareTitleDetails;
}
// {
//   "software_title": {
//     "id": 12,
//     "name": "Firefox.app",
//     "bundle_identifier": "org.mozilla.firefox",
//     "software_package": {
//       "name": "FalconSensor-6.44.pkg",
//       "version": "6.44",
//       "installer_id": 23,
//       "team_id": 3,
//       "uploaded_at": "2024-04-01T14:22:58Z",
//       "install_script": "sudo installer -pkg /temp/FalconSensor-6.44.pkg -target /",
//       "pre_install_query": "SELECT 1 FROM macos_profiles WHERE uuid='c9f4f0d5-8426-4eb8-b61b-27c543c9d3db';",
//       "post_install_script": "sudo /Applications/Falcon.app/Contents/Resources/falconctl license 0123456789ABCDEFGHIJKLMNOPQRSTUV-WX",
//       "self_service": true,
//       "status": {
//         "installed": 3,
//         "pending": 1,
//         "failed": 2,
//       }
//     },
//     "app_store_app": null,
//     "source": "apps", #ios_apps,ipados_apps
//     "browser": "",
//     "hosts_count": 48,
//     "versions": [
//       {
//         "id": 123,
//         "version": "117.0",
//         "vulnerabilities": ["CVE-2023-1234"],
//         "hosts_count": 37
//       },
//       {
//         "id": 124,
//         "version": "116.0",
//         "vulnerabilities": ["CVE-2023-4321"],
//         "hosts_count": 7
//       },
//       {
//         "id": 127,
//         "version": "115.5",
//         "vulnerabilities": ["CVE-2023-7654"],
//         "hosts_count": 4
//       }
//     ]
//   }
// }

// `GET /api/v1/fleet/software/versions/:id`
export interface ISoftwareVersionResponse {
  software: ISoftwareVersion;
}
// {
//   "software": {
//     "id": 425224,
//     "name": "Firefox.app",
//     "version": "117.0",
//     "bundle_identifier": "org.mozilla.firefox",
//     "source": "ipados_apps", # or ios_apps
//     "browser": "",
//     "generated_cpe": "cpe:2.3:a:mozilla:firefox:117.0:*:*:*:*:macos:*:*",
//     "vulnerabilities": [
//       {
//         "cve": "CVE-2023-4863",
//         "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2023-4863",
//         "created_at": "2024-07-01T00:15:00Z",
//         "cvss_score": 8.8, // Available in Fleet Premium
//         "epss_probability": 0.4101, // Available in Fleet Premium
//         "cisa_known_exploit": true, // Available in Fleet Premium
//         "cve_published": "2023-09-12T15:15:00Z", // Available in Fleet Premium
//         "resolved_in_version": "" // Available in Fleet Premium
//       },
//       {
//         "cve": "CVE-2023-5169",
//         "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2023-5169",
//         "created_at": "2024-07-01T00:15:00Z",
//         "cvss_score": 6.5, // Available in Fleet Premium
//         "epss_probability": 0.00073, // Available in Fleet Premium
//         "cisa_known_exploit": false, // Available in Fleet Premium
//         "cve_published": "2023-09-27T15:19:00Z", // Available in Fleet Premium
//         "resolved_in_version": "118" // Available in Fleet Premium
//       }
//     ]
//   }
// }

export interface ISoftwareVersionsQueryKey extends ISoftwareApiParams {
  scope: "software-versions";
}

export interface ISoftwareTitlesQueryKey extends ISoftwareApiParams {
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
    const path = teamId ? `${endpoint}?team_id=${teamId}` : endpoint;
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
    const path = teamId ? `${endpoint}?team_id=${teamId}` : endpoint;

    return sendRequest("GET", path);
  },

  addSoftwarePackage: (
    data: IAddSoftwareFormData,
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
    data.installScript && formData.append("install_script", data.installScript);
    data.preInstallCondition &&
      formData.append("pre_install_query", data.preInstallCondition);
    data.postInstallScript &&
      formData.append("post_install_script", data.postInstallScript);
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
