/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";

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
  operatingSystemId?: number;
}

export default {
  // hostCount.load share similar variables and parameters with hosts.loadAll
  load: (options: IHostCountLoadOptions | undefined) => {
    const { HOSTS_COUNT } = endpoints;
    const globalFilter = options?.globalFilter || "";
    const teamId = options?.teamId || null;
    const policyId = options?.policyId || null;
    const policyResponse = options?.policyResponse || null;
    const selectedLabels = options?.selectedLabels || [];
    const softwareId = options?.softwareId || null;
    const mdmId = options?.mdmId || null;
    const mdmEnrollmentStatus = options?.mdmEnrollmentStatus || null;
    const operatingSystemId = options?.operatingSystemId || null;

    const labelPrefix = "labels/";

    // Handle multiple filters
    const label = selectedLabels.find((f) => f.includes(labelPrefix));
    const status = selectedLabels.find((f) => !f.includes(labelPrefix));
    const isValidStatus =
      status === "new" || status === "online" || status === "offline";
    let queryString = "";

    if (globalFilter !== "") {
      queryString += `&query=${globalFilter}`;
    }

    if (status && isValidStatus) {
      queryString += `&status=${status}`;
    }

    if (label) {
      queryString += `&label_id=${parseInt(
        label.substr(labelPrefix.length),
        10
      )}`;
    }

    if (teamId) {
      queryString += `&team_id=${teamId}`;
    }

    if (!label && policyId) {
      queryString += `&policy_id=${policyId}`;
      queryString += `&policy_response=${policyResponse || "passing"}`; // TODO confirm whether there should be a default if there is an id but no response specified
    }

    // TODO: consider how to check for mutually exclusive scenarios with label, policy and software
    if (!label && !policyId && softwareId) {
      queryString += `&software_id=${softwareId}`;
    }

    if (!label && !policyId && mdmId) {
      queryString += `&mdm_id=${mdmId}`;
    }

    if (!label && !policyId && mdmEnrollmentStatus) {
      queryString += `&mdm_enrollment_status=${mdmEnrollmentStatus}`;
    }

    if (
      !label &&
      !policyId &&
      !softwareId &&
      !mdmId &&
      !mdmEnrollmentStatus &&
      operatingSystemId
    ) {
      queryString += `&operating_system_id=${operatingSystemId}`;
    }

    // Append query string to endpoint route after slicing off the leading ampersand
    const path = `${HOSTS_COUNT}${queryString && `?${queryString.slice(1)}`}`;

    return sendRequest("GET", path);
  },
};
