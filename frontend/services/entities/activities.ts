import sendRequest from "services";
import endpoints from "fleet/endpoints";

const DEFAULT_PAGE = 0;
const PER_PAGE = 8;
const ORDER_KEY = "created_at";
const ORDER_DIRECTION = "desc";

export default {
  loadNext: (page = DEFAULT_PAGE, perPage = PER_PAGE) => {
    const { ACTIVITIES } = endpoints;
    const pagination = `page=${page}&per_page=${perPage}`;
    const sort = `order_key=${ORDER_KEY}&order_direction=${ORDER_DIRECTION}`;
    const path = `${ACTIVITIES}?${pagination}&${sort}`;

    return sendRequest("GET", path);
  },
};
