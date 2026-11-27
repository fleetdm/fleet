/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */

import { IHost } from "interfaces/host";
import { ISelectedTargetsForApi } from "interfaces/target";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface ITargetsSearchParams {
  report_id?: number | null;
  query: string;
  excluded_host_ids: number[] | null;
  exclude_mobile_hosts?: boolean;
}

export interface ITargetsSearchResponse {
  hosts: IHost[];
}

export interface ITargetsCountParams {
  report_id?: number | null;
  selected: ISelectedTargetsForApi | null;
}

export interface ITargetsCountResponse {
  targets_count: number;
  targets_online: number;
  targets_offline: number;
}

export default {
  search: (params: ITargetsSearchParams): Promise<ITargetsSearchResponse> => {
    if (!params?.excluded_host_ids || !params?.query) {
      return Promise.reject("Invalid usage: missing required parameter(s)");
    }
    const { HOSTS } = endpoints;
    const path = `${HOSTS}/search`;

    return sendRequest("POST", path, params);
  },
  count: (params: ITargetsCountParams): Promise<ITargetsCountResponse> => {
    if (!params?.selected) {
      return Promise.reject("Invalid usage: no selected targets");
    }
    const { TARGETS } = endpoints;
    const path = `${TARGETS}/count`;

    return sendRequest("POST", path, params);
  },
};
