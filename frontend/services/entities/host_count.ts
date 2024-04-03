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
    // Order matches rest-api.md > List hosts parameters
    const status = options?.status;
    const query = options?.query || "";
    const policyId = options?.policyId;
    const policyResponse = options?.policyResponse;
    const teamId = options?.teamId;
    const softwareId = options?.softwareId;
    const softwareTitleId = options?.softwareTitleId;
    const softwareVersionId = options?.softwareVersionId;
    const labelId = options?.labelId;
    // TODO: Find out if where and how selectedFilters is being use
    const label = getLabelParam(options?.selectedLabels || []);
    const osName = options?.osName;
    const osVersionId = options?.osVersionId;
    const osVersion = options?.osVersion;
    const vulnerability = options?.vulnerability;
    const mdmId = options?.mdmId;
    const mdmEnrollmentStatus = options?.mdmEnrollmentStatus;
    const macSettingsStatus = options?.macSettingsStatus;
    const munkiIssueId = options?.munkiIssueId;
    const lowDiskSpaceHosts = options?.lowDiskSpaceHosts;
    const bootstrapPackageStatus = options?.bootstrapPackageStatus;
    const osSettings = options?.osSettings;
    const diskEncryptionStatus = options?.diskEncryptionStatus;

    const queryParams = {
      query,
      ...reconcileMutuallyInclusiveHostParams({
        teamId,
        macSettingsStatus,
        osSettings,
      }),
      ...reconcileMutuallyExclusiveHostParams({
        // Order matches rest-api.md > List hosts parameters
        policyId,
        policyResponse,
        softwareId,
        softwareTitleId,
        softwareVersionId,
        label,
        osName,
        osVersionId,
        osVersion,
        vulnerability,
        mdmId,
        mdmEnrollmentStatus,
        munkiIssueId,
        lowDiskSpaceHosts,
        bootstrapPackageStatus,
        diskEncryptionStatus,
        osSettings,
      }),
      label_id: labelId,
      status,
    };

    const queryString = buildQueryStringFromParams(queryParams);
    const endpoint = endpoints.HOSTS_COUNT;
    const path = `${endpoint}?${queryString}`;
    return sendRequest("GET", path);
  },
};
