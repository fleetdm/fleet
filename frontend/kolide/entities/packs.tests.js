import nock from "nock";

import Kolide from "kolide";
import mocks from "test/mocks";
import { hostStub, labelStub, packStub } from "test/stubs";

const { packs: packMocks } = mocks;

describe("Kolide - API client (packs)", () => {
  afterEach(() => {
    nock.cleanAll();
    Kolide.setBearerToken(null);
  });

  const bearerToken = "valid-bearer-token";

  describe("#addLabel", () => {
    it("calls the correct endpoint with the correct parameters", () => {
      const packID = 10;
      const labelID = 20;
      const request = packMocks.addLabel.valid(bearerToken, packID, labelID);

      Kolide.setBearerToken(bearerToken);
      return Kolide.packs.addLabel({ packID, labelID }).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#addQuery", () => {
    it("calls the correct endpoint with the correct parameters", () => {
      const packID = 10;
      const queryID = 20;
      const request = packMocks.addQuery.valid(bearerToken, packID, queryID);

      Kolide.setBearerToken(bearerToken);
      return Kolide.packs.addQuery({ packID, queryID }).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#create", () => {
    it("calls the correct endpoint with the correct parameters", () => {
      const { description, name } = packStub;
      const params = { description, name, host_ids: [], label_ids: [] };
      const request = packMocks.create.valid(bearerToken, params);

      Kolide.setBearerToken(bearerToken);

      return Kolide.packs.create(params).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#destroy", () => {
    it("calls the correct endpoint with the correct parameters", () => {
      const request = packMocks.destroy.valid(bearerToken, packStub);

      Kolide.setBearerToken(bearerToken);
      return Kolide.packs.destroy(packStub).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#update", () => {
    it("sends the host and/or label ids if packs are changed", () => {
      const label2 = { ...labelStub, id: 2 };
      const host2 = { ...hostStub, id: 2 };
      const pack = {
        ...packStub,
        host_ids: [host2.id],
        label_ids: [label2.id],
      };
      const targets = [host2, label2, hostStub, labelStub];
      const updatePackParams = {
        name: "New Pack Name",
        host_ids: [host2.id, hostStub.id],
        label_ids: [label2.id, labelStub.id],
      };
      const request = packMocks.update.valid(
        bearerToken,
        pack,
        updatePackParams
      );
      const updatedPack = { name: "New Pack Name", targets };

      Kolide.setBearerToken(bearerToken);
      return Kolide.packs.update(pack, updatedPack).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });
});
