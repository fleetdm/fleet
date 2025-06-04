/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
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
}

interface ILoadQueryReportQueryParams {
  order_key: string | undefined;
  order_direction: string | undefined;
  team_id?: number;
}

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

export default {
  load: ({ id, sortBy, teamId }: ILoadQueryReportOptions) => {
    const sortParams = getSortParams(sortBy);

    const queryParams: ILoadQueryReportQueryParams = {
      order_key: sortParams.order_key,
      order_direction: sortParams.order_direction,
    };
    if (teamId && teamId > 0) {
      queryParams.team_id = teamId;
    }

    const queryString = buildQueryStringFromParams(queryParams);

    const path = `${endpoints.QUERY_REPORT(id)}?${queryString}`;
    return sendRequest("GET", path);
  },
};
