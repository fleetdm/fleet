/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */

import { IQueryReport, IQueryReportResultRow } from "interfaces/query_report";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export interface ISortOption {
  key: string;
  direction: string;
}

export interface ILoadQueryReportOptions {
  id: number;
  sortBy: ISortOption[];
  teamId?: number;
  page?: number;
  perPage?: number;
  query?: string;
}

interface ILoadQueryReportQueryParams {
  order_key?: string;
  order_direction?: string;
  fleet_id?: number;
  page?: number;
  per_page?: number;
  query?: string;
}

/** Page size used when fetching every result, e.g. for CSV export. */
const LOAD_ALL_PAGE_SIZE = 5000;
/** Upper bound on pages fetched by loadAll, well above any report cap. */
const LOAD_ALL_MAX_PAGES = 1000;
export const LOAD_ALL_MAX_RESULTS = LOAD_ALL_PAGE_SIZE * LOAD_ALL_MAX_PAGES;

const getSortParams = (sortOptions?: ISortOption[]) => {
  if (sortOptions === undefined || sortOptions.length === 0) {
    return {};
  }

  const sortItem = sortOptions[0];
  return {
    order_key: sortItem.key,
    order_direction: sortItem.direction,
  };
};

const load = ({
  id,
  sortBy,
  teamId,
  page,
  perPage,
  query,
}: ILoadQueryReportOptions): Promise<IQueryReport> => {
  const sortParams = getSortParams(sortBy);

  const queryParams: ILoadQueryReportQueryParams = {
    order_key: sortParams.order_key,
    order_direction: sortParams.order_direction,
  };
  if (teamId && teamId > 0) {
    queryParams.fleet_id = teamId;
  }
  if (page !== undefined) {
    queryParams.page = page;
  }
  if (perPage !== undefined) {
    queryParams.per_page = perPage;
  }
  if (query) {
    queryParams.query = query;
  }

  const queryString = buildQueryStringFromParams(queryParams);

  const path = `${endpoints.QUERY_REPORT(id)}?${queryString}`;
  return sendRequest("GET", path);
};

/**
 * Fetches every matching result page by page, in the requested order.
 * Rejects rather than returning a partial set if the report is larger than
 * LOAD_ALL_MAX_RESULTS.
 */
const loadAll = async (
  options: Omit<ILoadQueryReportOptions, "page" | "perPage">
): Promise<IQueryReportResultRow[]> => {
  const results: IQueryReportResultRow[] = [];
  for (let page = 0; page < LOAD_ALL_MAX_PAGES; page += 1) {
    // Pages must be fetched in order, so awaiting inside the loop is intended.
    // eslint-disable-next-line no-await-in-loop
    const report = await load({
      ...options,
      page,
      perPage: LOAD_ALL_PAGE_SIZE,
    });
    results.push(...report.results);
    if (!report.meta?.has_next_results) {
      return results;
    }
  }
  throw new Error(
    `Report has more than ${LOAD_ALL_MAX_RESULTS} results; export is not supported.`
  );
};

export default {
  load,
  loadAll,
};
