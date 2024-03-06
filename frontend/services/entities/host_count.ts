/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import {
  buildQueryStringFromParams,
  getLabelParam,
  reconcileMutuallyExclusiveHostParams,
  reconcileMutuallyInclusiveHostParams,
} from "utilities/url";

import { IBaseHostsOptions } from "./hosts";

export interface ISortOption {
  key: string;
  direction: string;
}

export interface IHostsCountResponse {
  count: number;
}

export interface IHostsCountQueryKey extends IBaseHostsOptions {
  scope: "hosts_count";
}

export default {
  load: (
    options: IBaseHostsOptions | undefined
  ): Promise<IHostsCountResponse> => {
    const selectedLabels = options?.selectedLabels || [];
    const policyId = options?.policyId;
    const policyResponse = options?.policyResponse;
    const globalFilter = options?.globalFilter || "";
    const teamId = options?.teamId;
    const softwareId = options?.softwareId;
    const softwareTitleId = options?.softwareTitleId;
    const softwareVersionId = options?.softwareVersionId;
    const macSettingsStatus = options?.macSettingsStatus;
    const status = options?.status;
    const mdmId = options?.mdmId;
    const mdmEnrollmentStatus = options?.mdmEnrollmentStatus;
    const munkiIssueId = options?.munkiIssueId;
    const lowDiskSpaceHosts = options?.lowDiskSpaceHosts;
    const label = getLabelParam(selectedLabels);
    const osVersionId = options?.osVersionId;
    const osName = options?.osName;
    const osVersion = options?.osVersion;
    const osSettings = options?.osSettings;
    const vulnerability = options?.vulnerability;
    const diskEncryptionStatus = options?.diskEncryptionStatus;
    const bootstrapPackageStatus = options?.bootstrapPackageStatus;

    const queryParams = {
      query: globalFilter,
      ...reconcileMutuallyInclusiveHostParams({
        teamId,
        macSettingsStatus,
        osSettings,
      }),
      ...reconcileMutuallyExclusiveHostParams({
        label,
        policyId,
        policyResponse,
        mdmId,
        mdmEnrollmentStatus,
        munkiIssueId,
        softwareId,
        softwareTitleId,
        softwareVersionId,
        lowDiskSpaceHosts,
        osName,
        osVersionId,
        osVersion,
        osSettings,
        vulnerability,
        diskEncryptionStatus,
        bootstrapPackageStatus,
      }),
      label_id: label,
      status,
    };

    const queryString = buildQueryStringFromParams(queryParams);
    const endpoint = endpoints.HOSTS_COUNT;
    const path = `${endpoint}?${queryString}`;
    return sendRequest("GET", path);
  },
};
