import endpoints from "fleet/endpoints";
import { IActivity } from "interfaces/activity";
import sendRequest from "services";

const DEFAULT_PAGE = 0;
const PER_PAGE = 8;
const ORDER_KEY = "created_at";
const ORDER_DIRECTION = "desc";

export interface IActivitiesResponse {
  activities: IActivity[];
}

export default {
  loadNext: (
    page = DEFAULT_PAGE,
    perPage = PER_PAGE
  ): Promise<IActivitiesResponse> => {
    const { ACTIVITIES } = endpoints;
    const pagination = `page=${page}&per_page=${perPage}`;
    const sort = `order_key=${ORDER_KEY}&order_direction=${ORDER_DIRECTION}`;
    const path = `${ACTIVITIES}?${pagination}&${sort}`;

    return sendRequest("GET", path);
  },
};
