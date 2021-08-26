import nock from "nock";

import Fleet from "fleet";
import mocks from "test/mocks";
import { queryStub } from "test/stubs";

const { queries: queryMocks } = mocks;

describe("Kolide - API client (queries)", () => {
  afterEach(() => {
    nock.cleanAll();
    Fleet.setBearerToken(null);
  });

  const bearerToken = "valid-bearer-token";

  describe("#create", () => {
    it("calls the correct endpoint with the correct parameters", () => {
      const description = "query description";
      const name = "query name";
      const query = "SELECT * FROM users";
      const queryParams = { description, name, query };
      const request = queryMocks.create.valid(bearerToken, queryParams);

      Fleet.setBearerToken(bearerToken);
      return Fleet.queries.create(queryParams).then((queryResponse) => {
        expect(request.isDone()).toEqual(true);
        expect(queryResponse).toEqual(queryParams);
      });
    });
  });

  describe("#destroy", () => {
    it("calls the correct endpoint with the correct parameters", () => {
      const request = queryMocks.destroy.valid(bearerToken, queryStub);

      Fleet.setBearerToken(bearerToken);
      return Fleet.queries.destroy(queryStub).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#load", () => {
    it("calls the correct endpoint with the correct parameters", () => {
      const queryID = 10;
      const request = queryMocks.load.valid(bearerToken, queryID);

      Fleet.setBearerToken(bearerToken);
      return Fleet.queries.load(queryID).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#loadAll", () => {
    it("calls the correct endpoint with the correct parameters", () => {
      const request = queryMocks.loadAll.valid(bearerToken);

      Fleet.setBearerToken(bearerToken);
      return Fleet.queries.loadAll().then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#run", () => {
    it("calls the correct endpoint with the correct params", () => {
      const data = {
        query: "select * from users",
        selected: { hosts: [], labels: [] },
      };
      const request = queryMocks.run.valid(bearerToken, data);

      Fleet.setBearerToken(bearerToken);
      return Fleet.queries.run(data).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#update", () => {
    it("calls the correct endpoint with the correct params", () => {
      const query = {
        id: 1,
        name: "Query Name",
        description: "Query Description",
        query: "SELECT * FROM users",
      };
      const updateQueryParams = { name: "New Query Name" };
      const request = queryMocks.update.valid(
        bearerToken,
        query,
        updateQueryParams
      );

      Fleet.setBearerToken(bearerToken);
      return Fleet.queries
        .update(query, updateQueryParams)
        .then((queryResponse) => {
          expect(request.isDone()).toEqual(true);
          expect(queryResponse).toEqual({
            ...query,
            ...updateQueryParams,
          });
        });
    });
  });
});
