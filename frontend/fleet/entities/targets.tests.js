import nock from "nock";

import Fleet from "fleet";
import mocks from "test/mocks";

const { targets: targetMocks } = mocks;

describe("Kolide - API client (targets)", () => {
  afterEach(() => {
    nock.cleanAll();
    Fleet.setBearerToken(null);
  });

  const bearerToken = "valid-bearer-token";

  describe("#loadAll", () => {
    it("correctly parses the response", () => {
      nock.cleanAll();
      const hosts = [];
      const labels = [];
      const query = "mac";
      const queryId = 1;
      const request = targetMocks.loadAll.valid(bearerToken, query, queryId);

      Fleet.setBearerToken(bearerToken);
      return Fleet.targets
        .loadAll(query, queryId, { hosts, labels })
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });
});
