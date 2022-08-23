/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import {
  buildQueryStringFromParams,
  getLabelParam,
  reconcileMutuallyExclusiveHostParams,
  getStatusParam,
} from "utilities/url";

export interface ISortOption {
  key: string;
  direction: string;
}

export interface IHostCountLoadOptions {
  page?: number;
  perPage?: number;
  selectedLabels?: string[];
  globalFilter?: string;
  status?: string;
  teamId?: number;
  policyId?: number;
  policyResponse?: string;
  softwareId?: number;
  mdmId?: number;
  mdmEnrollmentStatus?: string;
  os_id?: number;
  os_name?: string;
  os_version?: string;
}

export default {
  load: (options: IHostCountLoadOptions | undefined) => {
    const selectedLabels = options?.selectedLabels || [];
    const policyId = options?.policyId;
    const policyResponse = options?.policyResponse;
    const globalFilter = options?.globalFilter || "";
    const teamId = options?.teamId;
    const softwareId = options?.softwareId;
    const mdmId = options?.mdmId;
    const mdmEnrollmentStatus = options?.mdmEnrollmentStatus;
    const label = getLabelParam(selectedLabels);

    const queryParams = {
      query: globalFilter,
      team_id: teamId,
      ...reconcileMutuallyExclusiveHostParams(
        label,
        policyId,
        policyResponse,
        mdmId,
        mdmEnrollmentStatus,
        softwareId
      ),
      status: getStatusParam(selectedLabels),
      label_id: label,
    };

    const queryString = buildQueryStringFromParams(queryParams);
    const endpoint = endpoints.HOSTS_COUNT;
    const path = `${endpoint}?${queryString}`;

    return sendRequest("GET", path);
  },
};
