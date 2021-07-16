import endpoints from "fleet/endpoints";

const DEFAULT_PAGE = 0;
const PER_PAGE = 10;

export default (client: any) => {
  return {
    loadNext: (page = DEFAULT_PAGE, perPage = PER_PAGE) => {
      const { ACTIVITIES } = endpoints;

      return client
        .authenticatedGet(client._endpoint(ACTIVITIES))
        .then((response: any) => response.activities);
    },
  };
};
