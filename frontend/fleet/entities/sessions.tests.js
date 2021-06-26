import nock from "nock";

import Fleet from "fleet";
import mocks from "test/mocks";

const { sessions: sessionMocks } = mocks;

describe("Kolide - API client (session)", () => {
  afterEach(() => {
    nock.cleanAll();
    Fleet.setBearerToken(null);
  });

  const bearerToken = "valid-bearer-token";

  describe("#create", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const params = { email: "admin@example.com", password: "secret" };
      const request = sessionMocks.create.valid(bearerToken, params);

      return Fleet.sessions.create(params).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#destroy", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const request = sessionMocks.destroy.valid(bearerToken);

      Fleet.setBearerToken(bearerToken);
      return Fleet.sessions.destroy().then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });
});
