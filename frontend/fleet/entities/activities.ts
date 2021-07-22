import endpoints from "fleet/endpoints";

const DEFAULT_PAGE = 0;
const PER_PAGE = 10;

export default (client: any): any => {
  return {
    loadNext: (page = DEFAULT_PAGE, perPage = PER_PAGE) => {
      const { ACTIVITIES } = endpoints;
      const pagination = `page=${page}&per_page=${perPage}`;
      const endpoint = `${ACTIVITIES}?${pagination}`;
      return client
        .authenticatedGet(client._endpoint(endpoint))
        .then((response: any) => response.activities);
    },
  };
};
