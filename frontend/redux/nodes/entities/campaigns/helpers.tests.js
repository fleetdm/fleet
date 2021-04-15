import helpers from "./helpers";

const host = {
  hostname: "jmeller-mbp.local",
  id: 1,
};
const campaign = {
  id: 4,
  query_id: 12,
  status: 0,
  user_id: 1,
  hosts_count: {
    successful: 0,
    failed: 0,
    total: 0,
  },
};
const campaignWithResults = {
  ...campaign,
  hosts: [{ id: 2, hostname: "some-machine" }],
  query_results: [
    { host: "some-machine", feature: "vendor", value: "GenuineIntel" },
  ],
  totals: {
    count: 3,
    online: 2,
  },
};
const { destroyFunc, updateCampaignState } = helpers;
const resultSocketData = {
  type: "result",
  data: {
    distributed_query_execution_id: 5,
    host,
    rows: [
      { feature: "product_name", value: "Intel Core" },
      { feature: "family", value: "0600" },
    ],
  },
};
const statusSocketData = {
  type: "status",
  data: "finished",
};
const totalsSocketData = {
  type: "totals",
  data: {
    count: 5,
    online: 1,
  },
};

describe("campaign entity - helpers", () => {
  describe("#destroyFunc", () => {
    it("returns the campaign", (done) => {
      destroyFunc(campaign)
        .then((response) => {
          expect(response).toEqual(campaign);
          done();
        })
        .catch(done);
    });
  });

  describe("#updateCampaignState", () => {
    it("appends query results to the campaign when the campaign has query results", () => {
      const state = { campaign: campaignWithResults };
      const updatedState = updateCampaignState(resultSocketData)(state, {});

      expect(updatedState.campaign.query_results).toEqual([
        ...campaignWithResults.query_results,
        { feature: "product_name", value: "Intel Core" },
        { feature: "family", value: "0600" },
      ]);
      expect(updatedState.campaign.hosts).toContainEqual(host);
    });

    it("adds query results to the campaign when the campaign does not have query results", () => {
      const state = { campaign };
      const updatedState = updateCampaignState(resultSocketData)(state, {});

      expect(updatedState.campaign.query_results).toEqual([
        { feature: "product_name", value: "Intel Core" },
        { feature: "family", value: "0600" },
      ]);
      expect(updatedState.campaign.hosts).toContainEqual(host);
    });

    it("updates totals on the campaign when the campaign has totals", () => {
      const state = { campaign: campaignWithResults };
      const updatedState = updateCampaignState(totalsSocketData)(state, {});

      expect(updatedState.campaign.totals).toEqual(totalsSocketData.data);
    });

    it("adds totals to the campaign when the campaign does not have totals", () => {
      const state = { campaign };
      const updatedState = updateCampaignState(totalsSocketData)(state, {});

      expect(updatedState.campaign.totals).toEqual(totalsSocketData.data);
    });

    it("increases the successful hosts count and total when the result has no error", () => {
      const state = { campaign };
      const updatedState = updateCampaignState(resultSocketData)(state, {});

      expect(updatedState.campaign.hosts_count).toEqual({
        successful: 1,
        failed: 0,
        total: 1,
      });
    });

    it("increases the failed hosts count and total when the result has an error", () => {
      const resultErrorSocketData = {
        type: "result",
        data: {
          ...resultSocketData.data,
          error: "failed",
        },
      };

      const state = { campaign };
      const updatedState = updateCampaignState(resultErrorSocketData)(
        state,
        {}
      );

      expect(updatedState.campaign.hosts_count).toEqual({
        successful: 0,
        failed: 1,
        total: 1,
      });
    });

    it("sets the queryIsRunning attribute for status socket data", () => {
      const state = { campaign };
      const updatedState = updateCampaignState(statusSocketData)(state, {});

      expect(updatedState.queryIsRunning).toEqual(false);
    });
  });
});
