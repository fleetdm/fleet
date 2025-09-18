import { IDeviceUserResponse } from "interfaces/host";
import { IListOptions } from "interfaces/list_options";
import { IDeviceSoftware, ISetupSoftwareStatus } from "interfaces/software";
import { IHostCertificate } from "interfaces/certificates";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import {
  buildQueryStringFromParams,
  getPathWithQueryParams,
} from "utilities/url";

import { IMdmCommandResult } from "interfaces/mdm";

import { createMockSetupSoftwareStatusesResponse } from "__mocks__/deviceUserMock";

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
  count: number;
}

export interface IGetDeviceCertsRequestParams extends IListOptions {
  token: string;
}

export interface IGetVppInstallCommandResultsResponse {
  results: IMdmCommandResult[];
}
export interface IGetSetupSoftwareStatusesResponse {
  setup_experience_results: { software?: ISetupSoftwareStatus[] };
}

export interface IGetSetupSoftwareStatusesParams {
  token: string;
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

    const path = getPathWithQueryParams(DEVICE_SOFTWARE(id), rest);
    return sendRequest("GET", path);
  },

  // getSoftwareIcon doesn't need its own service function because the logic is encapsulated in
  // softwareAPI.getSoftwareIconFromApiUrl in /entities/software.ts

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

  getSetupSoftwareStatuses: ({
    token,
  }: IGetSetupSoftwareStatusesParams): Promise<IGetSetupSoftwareStatusesResponse> => {
    const { DEVICE_SETUP_SOFTWARE_STATUSES } = endpoints;
    const path = DEVICE_SETUP_SOFTWARE_STATUSES(token);
    return sendRequest("POST", path);
  },
};
