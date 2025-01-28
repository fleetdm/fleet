/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { IHost, HostStatus } from "interfaces/host";
import {
  QueryParams,
  buildQueryStringFromParams,
  getLabelParam,
  reconcileMutuallyExclusiveHostParams,
  reconcileMutuallyInclusiveHostParams,
} from "utilities/url";
import {
  IHostSoftware,
  ISoftware,
  SoftwareAggregateStatus,
} from "interfaces/software";
import {
  DiskEncryptionStatus,
  BootstrapPackageStatus,
  IMdmSolution,
  MdmProfileStatus,
  MdmEnrollmentStatus,
} from "interfaces/mdm";
import { IMunkiIssuesAggregate } from "interfaces/macadmins";
import { PlatformValueOptions, PolicyResponse } from "utilities/constants";

export interface ISortOption {
  key: string;
  direction: string;
}

export interface ILoadHostsResponse {
  hosts: IHost[];
  software: ISoftware | undefined;
  software_title: { name: string; version?: string } | null | undefined; // TODO: confirm type
  munki_issue: IMunkiIssuesAggregate;
  mobile_device_management_solution: IMdmSolution;
}

export type IUnlockHostResponse =
  | {
      host_id: number;
      unlock_pin: string;
    }
  | Record<string, never>;

// the source of truth for the filter option names.
// there are used on many other pages but we define them here.
// TODO: add other filter options here.
export const HOSTS_QUERY_PARAMS = {
  OS_SETTINGS: "os_settings",
  DISK_ENCRYPTION: "os_settings_disk_encryption",
  SOFTWARE_STATUS: "software_status",
} as const;

export interface ILoadHostsQueryKey extends ILoadHostsOptions {
  scope: "hosts";
}

export type MacSettingsStatusQueryParam = "latest" | "pending" | "failing";

export interface ILoadHostsOptions {
  page?: number;
  perPage?: number;
  selectedLabels?: string[];
  globalFilter?: string;
  sortBy?: ISortOption[];
  teamId?: number;
  policyId?: number;
  policyResponse?: string;
  macSettingsStatus?: MacSettingsStatusQueryParam;
  softwareId?: number;
  softwareTitleId?: number;
  softwareVersionId?: number;
  softwareStatus?: SoftwareAggregateStatus;
  status?: HostStatus;
  mdmId?: number;
  mdmEnrollmentStatus?: string;
  lowDiskSpaceHosts?: number;
  osVersionId?: number;
  osName?: string;
  osVersion?: string;
  vulnerability?: string;
  munkiIssueId?: number;
  device_mapping?: boolean;
  columns?: string;
  visibleColumns?: string;
  osSettings?: MdmProfileStatus;
  diskEncryptionStatus?: DiskEncryptionStatus;
  bootstrapPackageStatus?: BootstrapPackageStatus;
}

export interface IExportHostsOptions {
  sortBy: ISortOption[];
  page?: number;
  perPage?: number;
  selectedLabels?: string[];
  globalFilter?: string;
  teamId?: number;
  policyId?: number;
  policyResponse?: string;
  macSettingsStatus?: MacSettingsStatusQueryParam;
  softwareId?: number;
  softwareTitleId?: number;
  softwareVersionId?: number;
  softwareStatus?: SoftwareAggregateStatus;
  status?: HostStatus;
  mdmId?: number;
  munkiIssueId?: number;
  mdmEnrollmentStatus?: string;
  lowDiskSpaceHosts?: number;
  osId?: number;
  osName?: string;
  osVersionId?: number;
  osVersion?: string;
  vulnerability?: string;
  device_mapping?: boolean;
  columns?: string;
  visibleColumns?: string;
  bootstrapPackageStatus?: BootstrapPackageStatus;
  osSettings?: MdmProfileStatus;
  diskEncryptionStatus?: DiskEncryptionStatus;
}

export interface IActionByFilter {
  teamId?: number | null;
  query: string;
  status: string;
  labelId?: number;
  currentTeam?: number | null;
  policyId?: number | null;
  policyResponse?: PolicyResponse;
  softwareId?: number | null;
  softwareTitleId?: number | null;
  softwareVersionId?: number | null;
  softwareStatus?: SoftwareAggregateStatus;
  osName?: string;
  osVersion?: string;
  osVersionId?: number | null;
  macSettingsStatus?: MacSettingsStatusQueryParam;
  bootstrapPackageStatus?: BootstrapPackageStatus;
  mdmId?: number | null;
  mdmEnrollmentStatus?: MdmEnrollmentStatus;
  munkiIssueId?: number | null;
  lowDiskSpaceHosts?: number | null;
  osSettings?: MdmProfileStatus;
  diskEncryptionStatus?: DiskEncryptionStatus;
  vulnerability?: string;
}

export interface IGetHostSoftwareResponse {
  software: IHostSoftware[];
  count: number;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export interface IHostSoftwareQueryParams extends QueryParams {
  page: number;
  per_page: number;
  query: string;
  order_key: string;
  order_direction: "asc" | "desc";
}

export interface IHostSoftwareQueryKey extends IHostSoftwareQueryParams {
  scope: "host_software";
  id: number;
  softwareUpdatedAt?: string;
}

export type ILoadHostDetailsExtension = "device_mapping" | "macadmins";

const LABEL_PREFIX = "labels/";

const getLabel = (selectedLabels?: string[]) => {
  if (selectedLabels === undefined) return undefined;
  return selectedLabels.find((filter) => filter.includes(LABEL_PREFIX));
};

const getHostEndpoint = (selectedLabels?: string[]) => {
  const { HOSTS, LABEL_HOSTS } = endpoints;
  if (selectedLabels === undefined) return HOSTS;

  const label = getLabel(selectedLabels);
  if (label) {
    const labelId = label.substr(LABEL_PREFIX.length);
    return LABEL_HOSTS(parseInt(labelId, 10));
  }

  return HOSTS;
};

const getSortParams = (sortOptions?: ISortOption[]) => {
  if (sortOptions === undefined || sortOptions.length === 0) {
    return {};
  }

  const sortItem = sortOptions[0];
  return {
    order_key: sortItem.key,
    order_direction: sortItem.direction,
  };
};

const createMdmParams = (platform?: PlatformValueOptions, teamId?: number) => {
  if (platform === "all") {
    return buildQueryStringFromParams({ team_id: teamId });
  }

  return buildQueryStringFromParams({ platform, team_id: teamId });
};

export default {
  destroy: (host: IHost) => {
    const { HOSTS } = endpoints;
    const path = `${HOSTS}/${host.id}`;

    return sendRequest("DELETE", path);
  },
  destroyBulk: (hostIds: number[]) => {
    const { HOSTS_DELETE } = endpoints;

    return sendRequest("POST", HOSTS_DELETE, { ids: hostIds });
  },
  destroyByFilter: ({
    teamId,
    query,
    status,
    labelId,
    policyId,
    policyResponse,
    softwareId,
    softwareTitleId,
    softwareVersionId,
    softwareStatus,
    osName,
    osVersion,
    osVersionId,
    macSettingsStatus,
    bootstrapPackageStatus,
    mdmId,
    mdmEnrollmentStatus,
    munkiIssueId,
    lowDiskSpaceHosts,
    osSettings,
    diskEncryptionStatus,
    vulnerability,
  }: IActionByFilter) => {
    const { HOSTS_DELETE } = endpoints;
    return sendRequest("POST", HOSTS_DELETE, {
      filters: {
        query: query || undefined, // Prevents empty string passed to API which as of 4.47 will return an error
        status,
        label_id: labelId,
        team_id: teamId,
        policy_id: policyId,
        policy_response: policyResponse,
        software_id: softwareId,
        software_title_id: softwareTitleId,
        software_version_id: softwareVersionId,
        [HOSTS_QUERY_PARAMS.SOFTWARE_STATUS]: softwareStatus,
        os_name: osName,
        os_version: osVersion,
        os_version_id: osVersionId,
        macos_settings: macSettingsStatus,
        bootstrap_package: bootstrapPackageStatus,
        mdm_id: mdmId,
        mdm_enrollment_status: mdmEnrollmentStatus,
        munki_issue_id: munkiIssueId,
        low_disk_space: lowDiskSpaceHosts,
        os_settings: osSettings,
        os_settings_disk_encryption: diskEncryptionStatus,
        vulnerability,
      },
    });
  },
  exportHosts: (options: IExportHostsOptions) => {
    const sortBy = options.sortBy;
    const selectedLabels = options?.selectedLabels || [];
    const globalFilter = options?.globalFilter || "";
    const teamId = options?.teamId;
    const policyId = options?.policyId;
    const policyResponse = options?.policyResponse || "passing";
    const softwareId = options?.softwareId;
    const softwareTitleId = options?.softwareTitleId;
    const softwareVersionId = options?.softwareVersionId;
    const softwareStatus = options?.softwareStatus;
    const macSettingsStatus = options?.macSettingsStatus;
    const osName = options?.osName;
    const osVersionId = options?.osVersionId;
    const osVersion = options?.osVersion;
    const status = options?.status;
    const mdmId = options?.mdmId;
    const mdmEnrollmentStatus = options?.mdmEnrollmentStatus;
    const lowDiskSpaceHosts = options?.lowDiskSpaceHosts;
    const bootstrapPackageStatus = options?.bootstrapPackageStatus;
    const visibleColumns = options?.visibleColumns;
    const label = getLabelParam(selectedLabels);
    const munkiIssueId = options?.munkiIssueId;
    const osSettings = options?.osSettings;
    const diskEncryptionStatus = options?.diskEncryptionStatus;
    const vulnerability = options?.vulnerability;

    if (!sortBy.length) {
      throw Error("sortBy is a required field.");
    }

    const queryParams = {
      order_key: sortBy[0].key,
      order_direction: sortBy[0].direction,
      query: globalFilter,
      ...reconcileMutuallyInclusiveHostParams({
        label,
        teamId,
        macSettingsStatus,
        osSettings,
      }),
      ...reconcileMutuallyExclusiveHostParams({
        teamId,
        label,
        policyId,
        policyResponse,
        mdmId,
        mdmEnrollmentStatus,
        munkiIssueId,
        softwareId,
        softwareTitleId,
        softwareVersionId,
        softwareStatus,
        osName,
        osVersionId,
        osVersion,
        lowDiskSpaceHosts,
        osSettings,
        bootstrapPackageStatus,
        diskEncryptionStatus,
        vulnerability,
      }),
      status,
      label_id: label,
      columns: visibleColumns,
      format: "csv",
    };

    const queryString = buildQueryStringFromParams(queryParams);
    const endpoint = endpoints.HOSTS_REPORT;
    const path = `${endpoint}?${queryString}`;

    return sendRequest("GET", path);
  },
  // TODO: change/remove this when backend implments way for client to get
  // a collection of hosts based on ho  st ids
  getHosts: (hostIds: number[]) => {
    return Promise.all(
      hostIds.map((hostId) => {
        const { HOSTS } = endpoints;
        const path = `${HOSTS}/${hostId}`;
        return sendRequest("GET", path);
      })
    );
  },

  loadHosts: ({
    page = 0,
    perPage = 100,
    globalFilter,
    teamId,
    policyId,
    policyResponse = "passing",
    macSettingsStatus,
    softwareId,
    softwareTitleId,
    softwareVersionId,
    softwareStatus,
    status,
    mdmId,
    mdmEnrollmentStatus,
    munkiIssueId,
    lowDiskSpaceHosts,
    osVersionId,
    osName,
    osVersion,
    vulnerability,
    device_mapping,
    selectedLabels,
    sortBy,
    osSettings,
    diskEncryptionStatus,
    bootstrapPackageStatus,
  }: ILoadHostsOptions): Promise<ILoadHostsResponse> => {
    const label = getLabel(selectedLabels);
    const sortParams = getSortParams(sortBy);

    const queryParams = {
      page,
      per_page: perPage,
      query: globalFilter,
      device_mapping,
      order_key: sortParams.order_key,
      order_direction: sortParams.order_direction,
      status,
      ...reconcileMutuallyInclusiveHostParams({
        label,
        teamId,
        macSettingsStatus,
        osSettings,
      }),
      ...reconcileMutuallyExclusiveHostParams({
        teamId,
        label,
        policyId,
        policyResponse,
        mdmId,
        mdmEnrollmentStatus,
        munkiIssueId,
        softwareId,
        softwareTitleId,
        softwareVersionId,
        softwareStatus,
        lowDiskSpaceHosts,
        osVersionId,
        osName,
        osVersion,
        vulnerability,
        diskEncryptionStatus,
        osSettings,
        bootstrapPackageStatus,
      }),
    };

    const queryString = buildQueryStringFromParams(queryParams);

    const endpoint = getHostEndpoint(selectedLabels);
    const path = `${endpoint}?${queryString}`;
    return sendRequest("GET", path);
  },
  loadHostDetails: (hostID: number) => {
    const { HOSTS } = endpoints;
    const path = `${HOSTS}/${hostID}?exclude_software=true`;

    return sendRequest("GET", path);
  },
  loadHostDetailsExtension: (
    hostID: number,
    extension: ILoadHostDetailsExtension
  ) => {
    const { HOSTS } = endpoints;
    const path = `${HOSTS}/${hostID}/${extension}`;

    return sendRequest("GET", path);
  },
  refetch: (host: IHost) => {
    const { HOSTS } = endpoints;
    const path = `${HOSTS}/${host.id}/refetch`;

    return sendRequest("POST", path);
  },
  search: (searchText: string) => {
    const { HOSTS } = endpoints;
    const path = `${HOSTS}?query=${searchText}`;

    return sendRequest("GET", path);
  },
  transferToTeam: (teamId: number | null, hostIds: number[]) => {
    const { HOSTS_TRANSFER } = endpoints;

    return sendRequest("POST", HOSTS_TRANSFER, {
      team_id: teamId,
      hosts: hostIds,
    });
  },

  // TODO confirm interplay with policies
  transferToTeamByFilter: ({
    teamId,
    query,
    status,
    labelId,
    currentTeam,
    policyId,
    policyResponse,
    softwareId,
    softwareTitleId,
    softwareVersionId,
    softwareStatus,
    osName,
    osVersion,
    osVersionId,
    macSettingsStatus,
    bootstrapPackageStatus,
    mdmId,
    mdmEnrollmentStatus,
    munkiIssueId,
    lowDiskSpaceHosts,
    osSettings,
    diskEncryptionStatus,
    vulnerability,
  }: IActionByFilter) => {
    const { HOSTS_TRANSFER_BY_FILTER } = endpoints;
    return sendRequest("POST", HOSTS_TRANSFER_BY_FILTER, {
      team_id: teamId,
      filters: {
        query: query || undefined, // Prevents empty string passed to API which as of 4.47 will return an error
        status,
        label_id: labelId,
        team_id: currentTeam,
        policy_id: policyId,
        policy_response: policyResponse,
        software_id: softwareId,
        software_title_id: softwareTitleId,
        software_version_id: softwareVersionId,
        [HOSTS_QUERY_PARAMS.SOFTWARE_STATUS]: softwareStatus,
        os_name: osName,
        os_version: osVersion,
        os_version_id: osVersionId,
        macos_settings: macSettingsStatus,
        bootstrap_package: bootstrapPackageStatus,
        mdm_id: mdmId,
        mdm_enrollment_status: mdmEnrollmentStatus,
        munki_issue_id: munkiIssueId,
        low_disk_space: lowDiskSpaceHosts,
        os_settings: osSettings,
        os_settings_disk_encryption: diskEncryptionStatus,
        vulnerability,
      },
    });
  },

  getMdm: (id: number) => {
    const { HOST_MDM } = endpoints;
    return sendRequest("GET", HOST_MDM(id));
  },

  getMdmSummary: (platform?: PlatformValueOptions, teamId?: number) => {
    const { MDM_SUMMARY } = endpoints;

    if (!platform || platform === "linux") {
      throw new Error("mdm not supported for this platform");
    }

    const params = createMdmParams(platform, teamId);
    const fullPath = params !== "" ? `${MDM_SUMMARY}?${params}` : MDM_SUMMARY;
    return sendRequest("GET", fullPath);
  },

  getEncryptionKey: (id: number) => {
    const { HOST_ENCRYPTION_KEY } = endpoints;
    return sendRequest("GET", HOST_ENCRYPTION_KEY(id));
  },

  lockHost: (id: number) => {
    const { HOST_LOCK } = endpoints;
    return sendRequest("POST", HOST_LOCK(id));
  },

  unlockHost: (id: number): Promise<IUnlockHostResponse> => {
    const { HOST_UNLOCK } = endpoints;
    return sendRequest("POST", HOST_UNLOCK(id));
  },

  wipeHost: (id: number) => {
    const { HOST_WIPE } = endpoints;
    return sendRequest("POST", HOST_WIPE(id));
  },

  resendProfile: (hostId: number, profileUUID: string) => {
    const { HOST_RESEND_PROFILE } = endpoints;

    return sendRequest("POST", HOST_RESEND_PROFILE(hostId, profileUUID));
  },

  getHostSoftware: (
    params: IHostSoftwareQueryKey
  ): Promise<IGetHostSoftwareResponse> => {
    const { HOST_SOFTWARE } = endpoints;
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { id, scope, ...rest } = params;
    const queryString = buildQueryStringFromParams(rest);

    return sendRequest("GET", `${HOST_SOFTWARE(id)}?${queryString}`);
  },

  installHostSoftwarePackage: (hostId: number, softwareId: number) => {
    const { HOST_SOFTWARE_PACKAGE_INSTALL } = endpoints;
    return sendRequest(
      "POST",
      HOST_SOFTWARE_PACKAGE_INSTALL(hostId, softwareId)
    );
  },
  uninstallHostSoftwarePackage: (hostId: number, softwareId: number) => {
    const { HOST_SOFTWARE_PACKAGE_UNINSTALL } = endpoints;
    return sendRequest(
      "POST",
      HOST_SOFTWARE_PACKAGE_UNINSTALL(hostId, softwareId)
    );
  },
};
