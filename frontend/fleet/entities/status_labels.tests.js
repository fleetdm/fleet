import nock from "nock";

import Fleet from "fleet";
import mocks from "test/mocks";

const { statusLabels: statusLabelMocks } = mocks;

describe("Kolide - API client (status labels)", () => {
  afterEach(() => {
    nock.cleanAll();
    Fleet.setBearerToken(null);
  });

  const bearerToken = "valid-bearer-token";

  describe("#getCounts", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const request = statusLabelMocks.getCounts.valid(bearerToken);

      Fleet.setBearerToken(bearerToken);
      return Fleet.statusLabels.getCounts().then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });
});
