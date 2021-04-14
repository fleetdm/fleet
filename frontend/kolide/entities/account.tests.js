import nock from "nock";

import Kolide from "kolide";
import mocks from "test/mocks";

const { account: accountMocks } = mocks;

describe("Kolide - API client (account)", () => {
  afterEach(() => {
    nock.cleanAll();
    Kolide.setBearerToken(null);
  });

  describe("#create", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const formData = {
        email: "hi@gnar.dog",
        name: "Gnar Dog",
        kolide_server_url: "https://gnar.kolide.co",
        org_logo_url: "https://thegnar.co/assets/logo.png",
        org_name: "The Gnar Co.",
        password: "p@ssw0rd",
        password_confirmation: "p@ssw0rd",
        username: "gnardog",
      };
      const request = accountMocks.create.valid(formData);

      return Kolide.account.create(formData).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });
});
