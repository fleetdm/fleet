import { IDeviceUserResponse } from "interfaces/host";
import { IListOptions } from "interfaces/list_options";
import { IDeviceSoftware } from "interfaces/software";
import { IHostCertificate } from "interfaces/certificates";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

import { IMdmCommandResult } from "interfaces/mdm";

import { IHostSoftwareQueryParams } from "./hosts";

export type ILoadHostDetailsExtension = "macadmins";

export interface IDeviceSoftwareQueryKey extends IHostSoftwareQueryParams {
  scope: "device_software";
  id: string;
  softwareUpdatedAt?: string;
}

export interface IGetDeviceSoftwareResponse {
  software: IDeviceSoftware[];
  count: number;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

interface IGetDeviceDetailsRequest {
  token: string;
  exclude_software?: boolean;
}

export interface IGetDeviceCertificatesResponse {
  certificates: IHostCertificate[];
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export interface IGetDeviceCertsRequestParams extends IListOptions {
  token: string;
}

export interface IGetVppInstallCommandResultsResponse {
  results: IMdmCommandResult[];
}

export default {
  loadHostDetails: ({
    token,
    exclude_software,
  }: IGetDeviceDetailsRequest): Promise<IDeviceUserResponse> => {
    const { DEVICE_USER_DETAILS } = endpoints;
    let path = `${DEVICE_USER_DETAILS}/${token}`;
    if (exclude_software) {
      path += "?exclude_software=true";
    }
    return sendRequest("GET", path);
  },
  loadHostDetailsExtension: (
    deviceAuthToken: string,
    extension: ILoadHostDetailsExtension
  ) => {
    const { DEVICE_USER_DETAILS } = endpoints;
    const path = `${DEVICE_USER_DETAILS}/${deviceAuthToken}/${extension}`;

    return sendRequest("GET", path);
  },
  refetch: (deviceAuthToken: string) => {
    const { DEVICE_USER_DETAILS } = endpoints;
    const path = `${DEVICE_USER_DETAILS}/${deviceAuthToken}/refetch`;

    return sendRequest("POST", path);
  },

  getDeviceSoftware: (
    params: IDeviceSoftwareQueryKey
  ): Promise<IGetDeviceSoftwareResponse> => {
    const { DEVICE_SOFTWARE } = endpoints;
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { id, scope, ...rest } = params;
    const queryString = buildQueryStringFromParams(rest);
    return sendRequest("GET", `${DEVICE_SOFTWARE(id)}?${queryString}`);
  },

  installSelfServiceSoftware: (
    deviceToken: string,
    softwareTitleId: number
  ) => {
    const { DEVICE_SOFTWARE_INSTALL } = endpoints;
    const path = DEVICE_SOFTWARE_INSTALL(deviceToken, softwareTitleId);

    return sendRequest("POST", path);
  },

  uninstallSelfServiceSoftware: (
    deviceToken: string,
    softwareTitleId: number
  ) => {
    const { DEVICE_SOFTWARE_UNINSTALL } = endpoints;
    return sendRequest(
      "POST",
      DEVICE_SOFTWARE_UNINSTALL(deviceToken, softwareTitleId)
    );
  },

  /** Gets more info on FMA/custom package install for device user */
  getSoftwareInstallResult: (deviceToken: string, uuid: string) => {
    const { DEVICE_SOFTWARE_INSTALL_RESULTS } = endpoints;
    const path = DEVICE_SOFTWARE_INSTALL_RESULTS(deviceToken, uuid);

    return sendRequest("GET", path);
  },

  /** Gets more info on FMA/custom package uninstall for device user */
  getSoftwareUninstallResult: (
    deviceToken: string,
    scriptExecutionId: string
  ) => {
    const { DEVICE_SOFTWARE_UNINSTALL_RESULTS } = endpoints;
    const path = DEVICE_SOFTWARE_UNINSTALL_RESULTS(
      deviceToken,
      scriptExecutionId
    );

    return sendRequest("GET", path);
  },

  /** Gets more info on VPP install for device user */
  getVppCommandResult: (
    deviceToken: string,
    uuid: string
  ): Promise<IGetVppInstallCommandResultsResponse> => {
    const { DEVICE_VPP_COMMAND_RESULTS } = endpoints;
    const path = DEVICE_VPP_COMMAND_RESULTS(deviceToken, uuid);

    return sendRequest("GET", path);
  },

  getDeviceCertificates: ({
    token,
    page,
    per_page,
    order_key,
    order_direction,
  }: IGetDeviceCertsRequestParams): Promise<IGetDeviceCertificatesResponse> => {
    const { DEVICE_CERTIFICATES } = endpoints;
    const path = `${DEVICE_CERTIFICATES(token)}?${buildQueryStringFromParams({
      page,
      per_page,
      order_key,
      order_direction,
    })}`;

    return sendRequest("GET", path);
  },
};
