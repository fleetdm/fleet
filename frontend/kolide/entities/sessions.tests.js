import nock from "nock";

import Kolide from "kolide";
import mocks from "test/mocks";

const { sessions: sessionMocks } = mocks;

describe("Kolide - API client (session)", () => {
  afterEach(() => {
    nock.cleanAll();
    Kolide.setBearerToken(null);
  });

  const bearerToken = "valid-bearer-token";

  describe("#create", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const params = { username: "admin", password: "secret" };
      const request = sessionMocks.create.valid(bearerToken, params);

      return Kolide.sessions.create(params).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#destroy", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const request = sessionMocks.destroy.valid(bearerToken);

      Kolide.setBearerToken(bearerToken);
      return Kolide.sessions.destroy().then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });
});
