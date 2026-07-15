import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export interface IChartDataPoint {
  timestamp: string;
  value: number;
}

export interface IChartFilters {
  label_ids?: number[];
  platforms?: string[];
  include_host_ids?: number[];
  exclude_host_ids?: number[];

  // CVE entity filters (cve metric only). Echoed back from the API.
  software_filters?: string[];
  has_known_exploit?: boolean;
  epss_min?: number;
  epss_max?: number;
  severity_min?: number;
  severity_max?: number;
  exclude_vulnerabilities?: string[];
}

export interface IChartResponse {
  metric: string;
  visualization: string;
  total_hosts: number;
  resolution: string;
  days: number;
  filters: IChartFilters;
  data: IChartDataPoint[];
}

export interface IChartApiParams {
  days?: number;
  resolution?: number;
  tz_offset?: number;
  fleet_id?: number;
  label_ids?: string;
  platforms?: string;
  include_host_ids?: string;
  exclude_host_ids?: string;

  // CVE entity filters (cve metric only). Lists are comma-separated; EPSS is
  // 0.0–1.0 (the Software tab converts from its 0–100 % input before sending).
  software_filters?: string;
  has_known_exploit?: boolean;
  epss_min?: number;
  epss_max?: number;
  severity_min?: number;
  severity_max?: number;
  exclude_vulnerabilities?: string;
}

export interface IChartQueryKey {
  scope: "chart";
  metric: string;
  params: IChartApiParams;
}

export default {
  getChartData: (metric: string, params: IChartApiParams = {}) => {
    const queryString = buildQueryStringFromParams(params);
    const endpoint = endpoints.CHART_DATA(metric);
    const path = queryString ? `${endpoint}?${queryString}` : endpoint;
    return sendRequest("GET", path);
  },
};
