import nock from "nock";

import Fleet from "fleet";
import { labelStub } from "test/stubs";
import mocks from "test/mocks";

const { labels: labelMocks } = mocks;

describe("Kolide - API client (labels)", () => {
  afterEach(() => {
    nock.cleanAll();
    Fleet.setBearerToken(null);
  });

  const bearerToken = "valid-bearer-token";

  describe("#create", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const description = "label description";
      const name = "label name";
      const platform = "windows";
      const query = "SELECT * FROM users";
      const id = 3;
      const labelParams = { description, name, platform, query, id };
      const request = labelMocks.create.valid(bearerToken, labelParams);

      Fleet.setBearerToken(bearerToken);
      return Fleet.labels.create(labelParams).then((labelResponse) => {
        expect(request.isDone()).toEqual(true);
        expect(labelResponse).toEqual({
          ...labelParams,
          display_text: name,
          slug: `labels/${id}`,
          type: "custom",
        });
      });
    });
  });

  describe("#destroy", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const request = labelMocks.destroy.valid(bearerToken, labelStub);

      Fleet.setBearerToken(bearerToken);
      return Fleet.labels.destroy(labelStub).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#update", () => {
    it("calls the appropriate endpoint with the correct parameters", () => {
      const params = { name: "New label name" };
      const request = labelMocks.update.valid(bearerToken, labelStub, params);

      Fleet.setBearerToken(bearerToken);
      return Fleet.labels
        .update(labelStub, params)
        .then(() => {
          expect(request.isDone()).toEqual(true);
        })
        .catch(() => {
          throw new Error("Request should have been stubbed");
        });
    });
  });
});
