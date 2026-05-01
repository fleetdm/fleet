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

export interface IChartRequestParams {
  days?: number;
  resolution?: number;
  tz_offset?: number;
  fleet_id?: number;
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

export default {
  getChartData: (metric: string, params: IChartRequestParams = {}) => {
    const queryString = buildQueryStringFromParams(params);
    const endpoint = endpoints.CHART_DATA(metric);
    const path = queryString ? `${endpoint}?${queryString}` : endpoint;
    return sendRequest("GET", path);
  },
};
