import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export interface IChartDataPoint {
  timestamp: string;
  values: Record<string, number>;
}

export interface ISeriesMeta {
  key: string;
  label: string;
  stats?: Record<string, any>;
}

export interface IChartFilters {
  label_ids?: number[];
  platforms?: string[];
  include_host_ids?: number[];
  exclude_host_ids?: number[];
}

export interface IChartResponse {
  metric: string;
  visualization: string;
  total_hosts: number;
  resolution: string;
  days: number;
  filters: IChartFilters;
  series: ISeriesMeta[];
  data: IChartDataPoint[];
}

export interface IChartRequestParams {
  days?: number;
  downsample?: number;
  tz_offset?: number;
  label_ids?: string;
  platforms?: string;
  include_host_ids?: string;
  exclude_host_ids?: string;
}

export interface IChartQueryKey {
  scope: "chart";
  metric: string;
  params: IChartRequestParams;
}

export interface IMostIgnoredPolicy {
  policy_id: number;
  name: string;
  team_id: number | null;
  team_name: string;
  failing_host_count: number;
}

export interface IMostIgnoredPoliciesResponse {
  policies: IMostIgnoredPolicy[];
}

export interface ITeamCompliance {
  team_id: number | null;
  name: string;
  host_count: number;
  hosts_failing_any: number;
  fully_compliant_pct: number;
  policies_tracked: number;
  policies_failing: number;
}

export interface IHostFailingSummary {
  host_id: number;
  hostname: string;
  computer_name: string;
  team_id: number | null;
  team_name: string;
  failing_policy_count: number;
}

export interface ITopNonCompliantHostsResponse {
  hosts: IHostFailingSummary[];
}

export default {
  getChartData: (metric: string, params: IChartRequestParams = {}) => {
    const queryString = buildQueryStringFromParams(params);
    const endpoint = endpoints.CHART_DATA(metric);
    const path = queryString ? `${endpoint}?${queryString}` : endpoint;
    return sendRequest("GET", path);
  },

  getMostIgnoredPolicies: (
    limit?: number
  ): Promise<IMostIgnoredPoliciesResponse> => {
    const queryString = buildQueryStringFromParams({ limit });
    const endpoint = endpoints.COMPLIANCE_MOST_IGNORED;
    const path = queryString ? `${endpoint}?${queryString}` : endpoint;
    return sendRequest("GET", path);
  },

  getTopNonCompliantHosts: (
    limit: number = 10
  ): Promise<ITopNonCompliantHostsResponse> => {
    const queryString = buildQueryStringFromParams({ limit });
    const endpoint = endpoints.COMPLIANCE_TOP_HOSTS;
    const path = queryString ? `${endpoint}?${queryString}` : endpoint;
    return sendRequest("GET", path);
  },
};
