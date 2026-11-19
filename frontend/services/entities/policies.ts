/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */

import { QueryablePlatform } from "interfaces/platform";
import {
  ILoadAllPoliciesResponse,
  IPoliciesCountResponse,
  IPolicyAutomationActivity,
  IPolicyFormData,
  IStoredPolicyResponse,
  PolicyAutomationActivityStatus,
  PolicyAutomationType,
  PolicyType,
} from "interfaces/policy";
import sendRequest from "services";
import {
  ListEntitiesResponseCommon,
  OrderDirection,
} from "services/entities/common";
import endpoints from "utilities/endpoints";
import {
  buildQueryStringFromParams,
  convertParamsToSnakeCase,
} from "utilities/url";

export interface IPoliciesFilterParams {
  /** Omitted for "All fleets", 0 for "Unassigned". */
  fleetId?: number;
  /** Adds the "All fleets" policies to a fleet's list or count. */
  mergeInherited?: boolean;
  query?: string;
  automationType?: PolicyAutomationType;
  type?: PolicyType;
  /** Targeted platform to filter policies by. */
  platform?: QueryablePlatform;
}

export interface IPoliciesListParams extends IPoliciesFilterParams {
  page?: number;
  perPage?: number;
  orderKey?: string;
  orderDirection?: OrderDirection;
}

export interface IPoliciesQueryKey extends IPoliciesListParams {
  scope: "policies";
}

export interface IPoliciesCountQueryKey extends IPoliciesFilterParams {
  scope: "policiesCount";
}

const ORDER_KEY = "name";
const ORDER_DIRECTION = "asc";

const filterQueryString = ({
  fleetId,
  mergeInherited,
  query,
  automationType,
  type,
  platform,
}: IPoliciesFilterParams) =>
  buildQueryStringFromParams(
    convertParamsToSnakeCase({
      fleetId,
      mergeInherited,
      query,
      automationType,
      type,
      platform,
    })
  );

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
  create: (data: IPolicyFormData): Promise<IStoredPolicyResponse> => {
    const {
      name,
      description,
      query,
      fleet_id,
      resolution,
      platform,
      critical,
      hidden,
      software_title_id,
      labels_include_any,
      labels_include_all,
      labels_exclude_any,
      labels_exclude_all,
      type,
      automation_type,
      continuous_automations_enabled,
      patch_when_closed,
      notify_before_patching,
      patch_deadline,
    } = data;

    return sendRequest("POST", endpoints.GLOBAL_POLICIES, {
      fleet_id: fleet_id ?? undefined,
      name,
      description,
      query,
      resolution,
      platform,
      critical,
      hidden,
      software_title_id,
      labels_include_any,
      labels_include_all,
      labels_exclude_any,
      labels_exclude_all,
      type,
      automation_type,
      continuous_automations_enabled,
      patch_when_closed,
      notify_before_patching,
      patch_deadline,
    });
  },

  // A policy keeps its fleet, so fleet_id isn't sent.
  update: (
    id: number,
    data: IPolicyFormData
  ): Promise<IStoredPolicyResponse> => {
    const {
      name,
      description,
      query,
      resolution,
      platform,
      critical,
      hidden,
      calendar_events_enabled,
      conditional_access_enabled,
      continuous_automations_enabled,
      patch_when_closed,
      notify_before_patching,
      patch_deadline,
      software_title_id,
      software_package_id,
      script_id,
      profile_uuid,
      labels_include_any,
      labels_include_all,
      labels_exclude_any,
      labels_exclude_all,
      automation_type,
    } = data;

    return sendRequest("PATCH", `${endpoints.GLOBAL_POLICIES}/${id}`, {
      name,
      description,
      query,
      resolution,
      platform,
      critical,
      hidden,
      calendar_events_enabled,
      conditional_access_enabled,
      continuous_automations_enabled,
      patch_when_closed,
      notify_before_patching,
      patch_deadline,
      software_title_id,
      software_package_id,
      script_id,
      profile_uuid,
      labels_include_any,
      labels_include_all,
      labels_exclude_any,
      labels_exclude_all,
      automation_type,
    });
  },

  /** Deletes policies from any scope in one request. */
  destroy: (ids: number[]) =>
    sendRequest("POST", `${endpoints.GLOBAL_POLICIES}/delete`, { ids }),

  loadAll: ({
    page,
    perPage,
    orderKey = ORDER_KEY,
    orderDirection = ORDER_DIRECTION,
    ...filters
  }: IPoliciesListParams): Promise<ILoadAllPoliciesResponse> => {
    const paging = buildQueryStringFromParams(
      convertParamsToSnakeCase({ page, perPage, orderKey, orderDirection })
    );
    const filter = filterQueryString(filters);
    const queryString = [paging, filter].filter(Boolean).join("&");

    return sendRequest("GET", `${endpoints.GLOBAL_POLICIES}?${queryString}`);
  },

  getCount: (
    filters: IPoliciesFilterParams
  ): Promise<IPoliciesCountResponse> => {
    const queryString = filterQueryString(filters);
    const path = `${endpoints.GLOBAL_POLICIES}/count`;

    return sendRequest("GET", queryString ? `${path}?${queryString}` : path);
  },

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

  reset: (id: number, hostId?: number): Promise<void> => {
    const { POLICY_RESET } = endpoints;
    const path =
      hostId === undefined
        ? POLICY_RESET(id)
        : `${POLICY_RESET(id)}?${buildQueryStringFromParams({
            host_id: hostId,
          })}`;

    return sendRequest("POST", path);
  },
};
