import endpoints from "utilities/endpoints";
import { IActivity } from "interfaces/activity";
import sendRequest from "services";

const DEFAULT_PAGE = 0;
const DEFAULT_PAGE_SIZE = 8;
const ORDER_KEY = "created_at";
const ORDER_DIRECTION = "desc";

export interface IActivitiesResponse {
  activities: IActivity[];
}

export default {
  loadNext: (
    page = DEFAULT_PAGE,
    perPage = DEFAULT_PAGE_SIZE
  ): Promise<IActivitiesResponse> => {
    const { ACTIVITIES } = endpoints;
    const pagination = `page=${page}&per_page=${perPage}`;
    const sort = `order_key=${ORDER_KEY}&order_direction=${ORDER_DIRECTION}`;
    const path = `${ACTIVITIES}?${pagination}&${sort}`;

    return sendRequest("GET", path);
  },
};
