/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */

import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";
import {
  IPolicyAutomationActivity,
  IStoredPolicyResponse,
  PolicyAutomationActivityStatus,
} from "interfaces/policy";
import {
  ListEntitiesResponseCommon,
  OrderDirection,
} from "services/entities/common";

export type PolicyAutomationActivitiesOrderKey =
  | "id"
  | "created_at"
  | "activity_type";

export interface IGetPolicyAutomationActivitiesParams {
  policyId: number;
  page?: number;
  perPage?: number;
  orderKey?: PolicyAutomationActivitiesOrderKey;
  orderDirection?: OrderDirection;
  query?: string;
  status?: PolicyAutomationActivityStatus | "";
}

export interface IPolicyAutomationActivitiesResponse
  extends ListEntitiesResponseCommon {
  activities: IPolicyAutomationActivity[];
}

export default {
  load: (id: number): Promise<IStoredPolicyResponse> => {
    const { GLOBAL_POLICIES } = endpoints;
    const path = `${GLOBAL_POLICIES}/${id}`;

    return sendRequest("GET", path);
  },

  getAutomationActivities: ({
    policyId,
    page,
    perPage,
    orderKey,
    orderDirection,
    query,
    status,
  }: IGetPolicyAutomationActivitiesParams): Promise<IPolicyAutomationActivitiesResponse> => {
    const { POLICY_AUTOMATION_ACTIVITIES } = endpoints;
    const queryString = buildQueryStringFromParams({
      page,
      per_page: perPage,
      order_key: orderKey,
      order_direction: orderDirection,
      query: query || undefined,
      status: status || undefined,
    });
    const path = `${POLICY_AUTOMATION_ACTIVITIES(policyId)}?${queryString}`;

    return sendRequest("GET", path);
  },

  reset: (id: number): Promise<void> => {
    const { POLICY_RESET } = endpoints;

    return sendRequest("POST", POLICY_RESET(id));
  },
};
