import { snakeCase, reduce } from "lodash";

import sendRequest from "services";
import endpoints from "utilities/endpoints";
import {
  ISoftwareResponse,
  ISoftwareCountResponse,
  IGetSoftwareByIdResponse,
} from "interfaces/software";
import { buildQueryStringFromParams, QueryParams } from "utilities/url";

interface IGetSoftwareProps {
  page?: number;
  perPage?: number;
  orderKey?: string;
  orderDir?: "asc" | "desc";
  query?: string;
  vulnerable?: boolean;
  teamId?: number;
}

type ISoftwareParams = Partial<IGetSoftwareProps>;

const ORDER_KEY = "name";
const ORDER_DIRECTION = "asc";

const convertParamsToSnakeCase = (params: ISoftwareParams) => {
  return reduce<typeof params, QueryParams>(
    params,
    (result, val, key) => {
      result[snakeCase(key)] = val;
      return result;
    },
    {}
  );
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
    const queryParams = {
      page,
      perPage,
      orderKey,
      orderDirection: orderDir,
      teamId,
      query,
      vulnerable,
    };

    const snakeCaseParams = convertParamsToSnakeCase(queryParams);
    const queryString = buildQueryStringFromParams(snakeCaseParams);
    const path = `${SOFTWARE}?${queryString}`;

    try {
      return sendRequest("GET", path);
    } catch (error) {
      throw error;
    }
  },

  count: async ({
    query,
    teamId,
    vulnerable,
  }: Pick<
    ISoftwareParams,
    "query" | "teamId" | "vulnerable"
  >): Promise<ISoftwareCountResponse> => {
    const { SOFTWARE } = endpoints;
    const path = `${SOFTWARE}/count`;
    const queryParams = {
      query,
      teamId,
      vulnerable,
    };
    const snakeCaseParams = convertParamsToSnakeCase(queryParams);
    const queryString = buildQueryStringFromParams(snakeCaseParams);

    return sendRequest("GET", path.concat(`?${queryString}`));
  },

  getSoftwareById: async (
    softwareId: string
  ): Promise<IGetSoftwareByIdResponse> => {
    const { SOFTWARE } = endpoints;
    const path = `${SOFTWARE}/${softwareId}`;

    return sendRequest("GET", path);
  },
};
