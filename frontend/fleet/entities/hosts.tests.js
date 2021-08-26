import nock from "nock";

import Fleet from "fleet";
import mocks from "test/mocks";
import createRequestMock from "test/mocks/create_request_mock";
import { hostStub } from "test/stubs";

const { hosts: hostMocks } = mocks;

describe("Kolide - API client (hosts)", () => {
  afterEach(() => {
    nock.cleanAll();
    Fleet.setBearerToken(null);
  });

  const bearerToken = "valid-bearer-token";

  describe("#destroy", () => {
    it("calls the correct endpoint with the correct params", () => {
      const request = hostMocks.destroy.valid(bearerToken, hostStub);

      Fleet.setBearerToken(bearerToken);
      return Fleet.hosts.destroy(hostStub).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#loadAll", () => {
    it("calls the correct endpoint with the correct query params when not passed any", () => {
      const request = hostMocks.loadAll.valid(bearerToken);

      Fleet.setBearerToken(bearerToken);
      return Fleet.hosts.loadAll().then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });

    it("calls the corret endpoint with the correct query params when passed multiple arguments", () => {
      const request = hostMocks.loadAll.validWithParams(bearerToken);
      const page = 3;
      const perPage = 100;
      const selectedFilter = "new";
      const query = "testQuery";
      const sortBy = [{ id: "hostname", direction: "desc" }];

      Fleet.setBearerToken(bearerToken);
      return Fleet.hosts
        .loadAll({
          page,
          perPage,
          selectedLabel: selectedFilter,
          globalFilter: query,
          sortBy,
        })
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });

    it("calls the label endpoint when used with label filter", () => {
      const request = createRequestMock({
        bearerToken,
        endpoint:
          "/api/v1/fleet/labels/6/hosts?page=2&per_page=50&order_key=hostname&order_direction=asc",
        method: "get",
        response: { hosts: [] },
      });

      Fleet.setBearerToken(bearerToken);
      return Fleet.hosts
        .loadAll({
          page: 2,
          perPage: 50,
          selectedLabel: "labels/6",
        })
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });
});
