import nock from "nock";

import Fleet from "fleet";
import mocks from "test/mocks";

const { account: accountMocks } = mocks;

describe("Kolide - API client (account)", () => {
  afterEach(() => {
    nock.cleanAll();
    Fleet.setBearerToken(null);
  });

  describe("#create", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const formData = {
        email: "hi@gnar.dog",
        name: "Gnar Dog",
        server_url: "https://gnar.Fleet.co",
        org_logo_url: "https://thegnar.co/assets/logo.png",
        org_name: "The Gnar Co.",
        password: "p@ssw0rd",
        password_confirmation: "p@ssw0rd",
      };
      const request = accountMocks.create.valid(formData);

      return Fleet.account.create(formData).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });
});
