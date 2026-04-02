import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export interface IHostReport {
  report_id: number;
  name: string;
  description: string;
  last_fetched: string | null;
  first_result: Record<string, string> | null;
  n_host_results: number;
  report_clipped: boolean;
  store_results: boolean;
}

export interface IListHostReportsResponse {
  reports: IHostReport[];
  count: number;
  meta: {
    has_previous_results: boolean;
    has_next_results: boolean;
  };
}

export interface IListHostReportsParams {
  page?: number;
  per_page?: number;
  order_key?: string;
  order_direction?: string;
  query?: string;
  include_reports_dont_store_results?: boolean;
}

export default {
  list: (
    hostId: number,
    params?: IListHostReportsParams
  ): Promise<IListHostReportsResponse> => {
    const { query, ...rest } = params || {};
    const queryParams: Record<string, string | number | boolean> = {};

    if (rest.page !== undefined) queryParams.page = rest.page;
    if (rest.per_page !== undefined) queryParams.per_page = rest.per_page;
    if (rest.order_key) queryParams.order_key = rest.order_key;
    if (rest.order_direction)
      queryParams.order_direction = rest.order_direction;
    if (query) queryParams.query = query;
    if (rest.include_reports_dont_store_results) {
      queryParams.include_reports_dont_store_results = true;
    }

    const queryString = buildQueryStringFromParams(queryParams);
    const path = `${endpoints.HOST_REPORTS(hostId)}?${queryString}`;
    return sendRequest("GET", path);
  },
};
