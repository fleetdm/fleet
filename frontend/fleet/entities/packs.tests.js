import nock from "nock";

import Fleet from "fleet";
import mocks from "test/mocks";
import { hostStub, labelStub, packStub } from "test/stubs";

const { packs: packMocks } = mocks;

describe("Kolide - API client (packs)", () => {
  afterEach(() => {
    nock.cleanAll();
    Fleet.setBearerToken(null);
  });

  const bearerToken = "valid-bearer-token";

  describe("#addLabel", () => {
    it("calls the correct endpoint with the correct parameters", () => {
      const packID = 10;
      const labelID = 20;
      const request = packMocks.addLabel.valid(bearerToken, packID, labelID);

      Fleet.setBearerToken(bearerToken);
      return Fleet.packs.addLabel({ packID, labelID }).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#addQuery", () => {
    it("calls the correct endpoint with the correct parameters", () => {
      const packID = 10;
      const queryID = 20;
      const request = packMocks.addQuery.valid(bearerToken, packID, queryID);

      Fleet.setBearerToken(bearerToken);
      return Fleet.packs.addQuery({ packID, queryID }).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#create", () => {
    it("calls the correct endpoint with the correct parameters", () => {
      const { description, name } = packStub;
      const params = {
        description,
        name,
        host_ids: [],
        label_ids: [],
        team_ids: [],
      };
      const request = packMocks.create.valid(bearerToken, params);

      Fleet.setBearerToken(bearerToken);

      return Fleet.packs.create(params).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });

  describe("#destroy", () => {
    it("calls the correct endpoint with the correct parameters", () => {
      const request = packMocks.destroy.valid(bearerToken, packStub);

      Fleet.setBearerToken(bearerToken);
      return Fleet.packs.destroy(packStub).then(() => {
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
        team_ids: [],
      };
      const request = packMocks.update.valid(
        bearerToken,
        pack,
        updatePackParams
      );
      const updatedPack = { name: "New Pack Name", targets };

      Fleet.setBearerToken(bearerToken);
      return Fleet.packs.update(pack, updatedPack).then(() => {
        expect(request.isDone()).toEqual(true);
      });
    });
  });
});
