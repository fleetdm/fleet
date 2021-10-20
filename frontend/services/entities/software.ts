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

const DEFAULT_PAGE = 0;
const PER_PAGE = 8;
const ORDER_KEY = "hosts_count";
const ORDER_DIRECTION = "desc";

export default {
  load: async ({
    page = DEFAULT_PAGE,
    perPage = PER_PAGE,
    orderKey = ORDER_KEY,
    orderDir = ORDER_DIRECTION,
    query,
    vulnerable,
    teamId,
  }: ISoftwareParams): Promise<ISoftware[]> => {
    const { SOFTWARE } = endpoints;
    const pagination = `page=${page}&per_page=${perPage}`;
    const sort = `order_key=${orderKey}&order_direction=${orderDir}`;
    const team = teamId ? `team_id=${teamId}` : "";
    const path = `${SOFTWARE}?${pagination}&${sort}&${team}&${query}&${vulnerable}`;

    try {
      const { software }: ISoftwareResponse = await sendRequest("GET", path);
      return software;
    } catch (error) {
      throw error;
    }
  },
};
