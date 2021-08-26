import Fleet from "fleet";
import mocks from "test/mocks";
import { userStub } from "test/stubs";

const { invites: inviteMocks } = mocks;

describe("Kolide - API client (invites)", () => {
  afterEach(() => {
    Fleet.setBearerToken(null);
  });

  const bearerToken = "valid-bearer-token";

  describe("#create", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const formData = {
        email: "new@user.org",
        admin: false,
        invited_by: 1,
        id: 1,
        name: "",
      };
      const request = inviteMocks.create.valid(bearerToken, formData);

      Fleet.setBearerToken(bearerToken);
      return Fleet.invites.create(formData).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#destroy", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const request = inviteMocks.destroy.valid(bearerToken, userStub);

      Fleet.setBearerToken(bearerToken);
      return Fleet.invites.destroy(userStub).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#loadAll", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const request = inviteMocks.loadAll.valid(bearerToken);

      Fleet.setBearerToken(bearerToken);
      return Fleet.invites.loadAll().then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });

    it("calls the appropriate endpoint with the correct query params when passed multiple arguments", () => {
      const request = inviteMocks.loadAll.validWithParams(bearerToken);
      const page = 3;
      const perPage = 100;
      const query = "testQuery";
      const sortBy = [{ id: "name", desc: true }];

      Fleet.setBearerToken(bearerToken);
      return Fleet.invites.loadAll(page, perPage, query, sortBy).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });
});
