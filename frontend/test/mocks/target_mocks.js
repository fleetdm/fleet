import createRequestMock from "test/mocks/create_request_mock";

export default {
  loadAll: {
    valid: (bearerToken, query, queryId) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/v1/fleet/targets",
        method: "post",
        params: {
          query,
          query_id: queryId,
          selected: {
            hosts: [],
            labels: [],
          },
        },
        response: {
          targets_count: 1234,
          targets: [
            {
              id: 3,
              label: "OS X El Capitan 10.11",
              name: "osx-10.11",
              platform: "darwin",
              target_type: "hosts",
            },
            {
              id: 4,
              label: "Jason Meller's Macbook Pro",
              name: "jmeller.local",
              platform: "darwin",
              target_type: "hosts",
            },
            {
              id: 4,
              label: "All Macs",
              name: "macs",
              count: 1234,
              target_type: "labels",
            },
          ],
        },
      });
    },
  },
};
