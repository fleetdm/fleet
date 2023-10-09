/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
// import sendRequest from "services";
import endpoints from "utilities/endpoints";

import { buildQueryStringFromParams } from "utilities/url";

// Mock API requests to be used in developing FE for #7766 in parallel with BE development
import { sendRequest } from "services/mock_service/service/service";

export interface ISortOption {
  key: string;
  direction: string;
}

export interface ILoadQueryReportOptions {
  id: number;
  sortBy: ISortOption[];
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
  load: ({ id, sortBy }: ILoadQueryReportOptions) => {
    const sortParams = getSortParams(sortBy);

    const { QUERIES } = endpoints;

    const queryParams = {
      order_key: sortParams.order_key,
      order_direction: sortParams.order_direction,
    };

    const queryString = buildQueryStringFromParams(queryParams);

    // const endpoint = `${QUERIES}/${id}/report`;
    const endpoint = `${QUERIES}/113/report`;
    const path = `${endpoint}?${queryString}`;
    return sendRequest("GET", path);
  },
};
