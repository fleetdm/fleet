import nock from "nock";

import Fleet from "fleet";
import mocks from "test/mocks";
import { globalScheduledQueryStub } from "test/stubs";

const { globalScheduledQueries: globalScheduledQueryMocks } = mocks;

describe("Fleet - API client (global scheduled queries)", () => {
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
        platform: "darwin",
        query_id: 2,
        shard: 12,
      };
      const request = globalScheduledQueryMocks.create.valid(
        bearerToken,
        formData
      );

      Fleet.setBearerToken(bearerToken);
      return Fleet.globalScheduledQueries.create(formData).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#destroy", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const scheduledQuery = { id: 1 };
      const request = globalScheduledQueryMocks.destroy.valid(
        bearerToken,
        scheduledQuery
      );
      Fleet.setBearerToken(bearerToken);
      return Fleet.globalScheduledQueries.destroy(scheduledQuery).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#loadAll", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const request = globalScheduledQueryMocks.loadAll.valid(bearerToken);

      Fleet.setBearerToken(bearerToken);
      return Fleet.globalScheduledQueries.loadAll().then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#update", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const updatedAttrs = { interval: 200 };
      const request = globalScheduledQueryMocks.update.valid(
        bearerToken,
        globalScheduledQueryStub,
        updatedAttrs
      );

      Fleet.setBearerToken(bearerToken);
      return Fleet.globalScheduledQueries
        .update(globalScheduledQueryStub, updatedAttrs)
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });
});
