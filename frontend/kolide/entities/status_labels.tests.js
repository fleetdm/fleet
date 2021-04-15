import nock from "nock";

import Kolide from "kolide";
import mocks from "test/mocks";

const { statusLabels: statusLabelMocks } = mocks;

describe("Kolide - API client (status labels)", () => {
  afterEach(() => {
    nock.cleanAll();
    Kolide.setBearerToken(null);
  });

  const bearerToken = "valid-bearer-token";

  describe("#getCounts", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const request = statusLabelMocks.getCounts.valid(bearerToken);

      Kolide.setBearerToken(bearerToken);
      return Kolide.statusLabels.getCounts().then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });
});
