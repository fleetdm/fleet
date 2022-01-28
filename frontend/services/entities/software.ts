import { snakeCase } from "lodash";

import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { ISoftware } from "interfaces/software";

interface IGetSoftwareProps {
  page: number;
  perPage?: number;
  orderKey: string;
  orderDir: "asc" | "desc";
  query: string;
  vulnerable: boolean;
  teamId?: number;
}

export interface ISoftwareResponse {
  counts_updated_at: string;
  software: ISoftware[];
}

export interface ISoftwareCountResponse {
  count: number;
}

type ISoftwareParams = Partial<IGetSoftwareProps>;

const ORDER_KEY = "name";
const ORDER_DIRECTION = "asc";

const buildQueryStringFromParams = (params: ISoftwareParams) => {
  const filteredParams = Object.entries(params).filter(
    ([key, value]) => !!value
  );
  if (!filteredParams.length) {
    return "";
  }
  return `?${filteredParams
    .map(
      ([key, value]) =>
        `${encodeURIComponent(snakeCase(key))}=${encodeURIComponent(value)}`
    )
    .join("&")}`;
};

export default {
  load: async ({
    page,
    perPage,
    orderKey = ORDER_KEY,
    orderDir = ORDER_DIRECTION,
    query,
    vulnerable,
    teamId,
  }: ISoftwareParams): Promise<ISoftwareResponse> => {
    const { SOFTWARE } = endpoints;
    const pagination = perPage ? `page=${page}&per_page=${perPage}` : "";
    const sort = `order_key=${orderKey}&order_direction=${orderDir}`;
    let path = `${SOFTWARE}?${pagination}&${sort}`;

    if (teamId) {
      path += `&team_id=${teamId}`;
    }

    if (query) {
      path += `&query=${encodeURIComponent(query)}`;
    }

    if (vulnerable) {
      path += `&vulnerable=${vulnerable}`;
    }

    try {
      return sendRequest("GET", path);
    } catch (error) {
      throw error;
    }
  },

  count: async (params: ISoftwareParams): Promise<ISoftwareCountResponse> => {
    const { SOFTWARE } = endpoints;
    const path = `${SOFTWARE}/count`;
    const queryString = buildQueryStringFromParams(params);

    return sendRequest("GET", path.concat(queryString));
  },
};
