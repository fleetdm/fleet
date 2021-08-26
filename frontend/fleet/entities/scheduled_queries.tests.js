import nock from "nock";

import Fleet from "fleet";
import mocks from "test/mocks";
import { scheduledQueryStub } from "test/stubs";

const { scheduledQueries: scheduledQueryMocks } = mocks;

describe("Kolide - API client (scheduled queries)", () => {
  afterEach(() => {
    nock.cleanAll();
    Fleet.setBearerToken(null);
  });

  const bearerToken = "valid-bearer-token";

  describe("#create", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const formData = {
        interval: 60,
        logging_type: "differential",
        pack_id: 1,
        platform: "darwin",
        query_id: 2,
        shard: 12,
      };
      const request = scheduledQueryMocks.create.valid(bearerToken, formData);

      Fleet.setBearerToken(bearerToken);
      return Fleet.scheduledQueries.create(formData).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#destroy", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const scheduledQuery = { id: 1 };
      const request = scheduledQueryMocks.destroy.valid(
        bearerToken,
        scheduledQuery
      );

      Fleet.setBearerToken(bearerToken);
      return Fleet.scheduledQueries.destroy(scheduledQuery).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#loadAll", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const pack = { id: 1 };
      const request = scheduledQueryMocks.loadAll.valid(bearerToken, pack);

      Fleet.setBearerToken(bearerToken);
      return Fleet.scheduledQueries.loadAll(pack).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#update", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const updatedAttrs = { interval: 200 };
      const request = scheduledQueryMocks.update.valid(
        bearerToken,
        scheduledQueryStub,
        updatedAttrs
      );

      Fleet.setBearerToken(bearerToken);
      return Fleet.scheduledQueries
        .update(scheduledQueryStub, updatedAttrs)
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });
});
