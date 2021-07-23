import endpoints from "fleet/endpoints";

const DEFAULT_PAGE = 0;
const PER_PAGE = 10;
const ORDER_KEY = "created_at";
const ORDER_DIRECTION = "desc";

export default (client: any): any => {
  return {
    loadNext: (page = DEFAULT_PAGE, perPage = PER_PAGE) => {
      const { ACTIVITIES } = endpoints;
      const pagination = `page=${page}&per_page=${perPage}`;
      const sort = `order_key=${ORDER_KEY}&order_direction=${ORDER_DIRECTION}`;
      const endpoint = `${ACTIVITIES}?${pagination}&${sort}`;

      return client
        .authenticatedGet(client._endpoint(endpoint))
        .then((response: any) => {
          console.log("activities response: ", response);
          return response.activities;
        })
        .catch((err: any) => {
          console.log("activities err: ", err);
          return new Error(err);
        });
    },
  };
};
