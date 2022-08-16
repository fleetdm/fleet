/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

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

const getPolicyParams = (
  label?: string,
  policyId?: number,
  policyResponse?: string
) => {
  if (label !== undefined || policyId === undefined) return {};

  return {
    policy_id: policyId,
    policy_response: policyResponse,
  };
};

const getSoftwareParam = (
  label?: string,
  policyId?: number,
  softwareId?: number,
  mdmId?: number,
  mdmEnrollmentStatus?: string
) => {
  return !label && !policyId && !mdmId && !mdmEnrollmentStatus
    ? softwareId
    : undefined;
};

const getMDMParams = (
  label?: string,
  policyId?: number,
  softwareId?: number,
  mdmId?: number,
  mdmEnrollmentStatus?: string
) => {
  if (!label && !policyId && !softwareId && !mdmEnrollmentStatus && !mdmId)
    return undefined;

  return { mdmId: mdmId, mdmEnrollmentStatus: mdmEnrollmentStatus };
};

const getOperatingSystemParam = (
  label?: string,
  policyId?: number,
  softwareId?: number,
  operatingSystemId?: number
) => {
  return label === undefined &&
    policyId === undefined &&
    softwareId === undefined
    ? operatingSystemId
    : undefined;
};

const LABEL_PREFIX = "labels/";

const getStatusParam = (selectedLabels?: string[]) => {
  if (selectedLabels === undefined) return undefined;

  const status = selectedLabels.find((f) => !f.includes(LABEL_PREFIX));
  if (status === undefined) return undefined;

  const statusFilterList = ["new", "online", "offline"];
  return statusFilterList.includes(status) ? status : undefined;
};

const getLabelParam = (selectedLabels?: string[]) => {
  if (selectedLabels === undefined) return undefined;

  const label = selectedLabels.find((f) => f.includes(LABEL_PREFIX));
  if (label === undefined) return undefined;

  return label.slice(7);
};

export default {
  // hostCount.load share similar variables and parameters with hosts.loadAll
  load: (options: IHostCountLoadOptions | undefined) => {
    const selectedLabels = options?.selectedLabels || [];
    const policyId = options?.policyId;
    const policyResponse = options?.policyResponse;
    const globalFilter = options?.globalFilter || "";
    const teamId = options?.teamId || null;
    const softwareId = options?.softwareId;
    const mdmId = options?.mdmId;
    const mdmEnrollmentStatus = options?.mdmEnrollmentStatus;
    const operatingSystemId = options?.operatingSystemId;
    const label = getLabelParam(selectedLabels);
    const policyParams = getPolicyParams(label, policyId, policyResponse);
    const mdmParams = getMDMParams(
      label,
      policyId,
      softwareId,
      mdmId,
      mdmEnrollmentStatus
    );

    const queryParams = {
      query: globalFilter,
      team_id: teamId,
      policy_id: policyParams.policy_id,
      policy_response: policyParams.policy_response,
      software_id: getSoftwareParam(label, policyId, softwareId),
      mdm_id: mdmParams?.mdmId,
      mdm_enrollment_status: mdmParams?.mdmEnrollmentStatus,
      operating_system_id: getOperatingSystemParam(
        label,
        policyId,
        softwareId,
        operatingSystemId
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
