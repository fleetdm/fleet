import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { ISoftware } from "interfaces/software";

interface IGetSoftwareProps {
  page: number;
  perPage: number;
  orderKey: string;
  orderDir: "asc" | "desc";
  query: string;
  vulnerable: boolean;
  teamId: boolean;
}

interface ISoftwareResponse {
  software: ISoftware[];
}

type ISoftwareParams = Partial<IGetSoftwareProps>;

const ORDER_KEY = "name";
const ORDER_DIRECTION = "asc";

export default {
  load: async ({
    page,
    perPage,
    orderKey = ORDER_KEY,
    orderDir = ORDER_DIRECTION,
    query,
    vulnerable,
    teamId,
  }: ISoftwareParams): Promise<ISoftware[]> => {
    const { SOFTWARE } = endpoints;
    const pagination = `page=${page}&per_page=${perPage}`;
    const sort = `order_key=${orderKey}&order_direction=${orderDir}`;
    let path = `${SOFTWARE}?${pagination}&${sort}`;

    if (teamId) {
      path += `&team_id=${teamId}`;
    }

    if (query) {
      path += `&query=${query}`;
    }

    if (vulnerable) {
      path += `&vulnerable=${vulnerable}`;
    }

    try {
      const { software }: ISoftwareResponse = await sendRequest("GET", path);
      return software;
    } catch (error) {
      throw error;
    }
  },
};
