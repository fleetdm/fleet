import React from "react";
import { keys } from "lodash";
import { mount } from "enzyme";

import { fillInFormInput } from "test/helpers";
import QueryResultsTable from "components/queries/QueryResultsTable";

const host = {
  detail_updated_at: "2016-10-25T16:24:27.679472917-04:00",
  hostname: "jmeller-mbp.local",
  id: 1,
  ip: "192.168.1.10",
  mac: "10:11:12:13:14:15",
  memory: 4145483776,
  os_version: "Mac OS X 10.11.6",
  osquery_version: "2.0.0",
  platform: "darwin",
  status: "online",
  updated_at: "0001-01-01T00:00:00Z",
  uptime: 3600000000000,
  uuid: "1234-5678-9101",
};
const queryResult = {
  distributed_query_execution_id: 4,
  host,
  rows: [{ host_hostname: "dfoihgsx", cwd: "/", directory: "/root" }],
};

const campaignWithNoQueryResults = {
  created_at: "0001-01-01T00:00:00Z",
  hosts_count: {
    failed: 0,
    successful: 0,
    total: 0,
  },
  id: 4,
  query_id: 12,
  status: 0,
  totals: {
    count: 3,
    online: 2,
  },
  updated_at: "0001-01-01T00:00:00Z",
  user_id: 1,
};
const campaignWithQueryResults = {
  ...campaignWithNoQueryResults,
  query_results: [
    { host_hostname: "dfoihgsx", cwd: "/", directory: "/root" },
    { host_hostname: "abc123", cwd: "/", directory: "/root" },
  ],
  Metrics: {
    OnlineHosts: 2,
    OfflineHosts: 0,
  },
  hosts_count: {
    failed: 0,
    successful: 2,
    total: 2,
  },
};

describe("QueryResultsTable - component", () => {
  const componentWithoutQueryResults = mount(
    <QueryResultsTable campaign={campaignWithNoQueryResults} />
  );
  const componentWithQueryResults = mount(
    <QueryResultsTable campaign={campaignWithQueryResults} />
  );

  it("renders", () => {
    expect(componentWithoutQueryResults.length).toEqual(1);
    expect(componentWithQueryResults.length).toEqual(1);
  });

  it("renders a QueryProgressDetails component if Results is Fullscreen", () => {
    const component = mount(
      <QueryResultsTable
        campaign={campaignWithQueryResults}
        isQueryFullScreen
      />
    );
    const QueryProgressDetails = component.find("QueryProgressDetails");

    expect(QueryProgressDetails.length).toEqual(
      1,
      "QueryProgressDetails did not render"
    );
  });

  it("doesn't render a QueryProgressDetails component if Results isn't Fullscreen", () => {
    const QueryProgressDetails = componentWithQueryResults.find(
      "QueryProgressDetails"
    );

    expect(QueryProgressDetails.length).toEqual(
      0,
      "QueryProgressDetails did not render"
    );
  });

  it("sets the column headers to the keys of the query results", () => {
    const queryResultKeys = keys(queryResult.rows[0]);
    const tableHeaderText = componentWithQueryResults.find("thead").text();

    queryResultKeys.forEach((key) => {
      if (key === "host_hostname") {
        expect(tableHeaderText).toContain("hostname");
      } else {
        expect(tableHeaderText).toContain(key);
      }
    });
  });

  it("filters by hostname", () => {
    const hostnameInputFilter = componentWithQueryResults
      .find("InputField")
      .find({ name: "hostname" });

    expect(componentWithQueryResults.find("QueryResultsRow").length).toEqual(2);

    fillInFormInput(hostnameInputFilter, "abc123");

    expect(componentWithQueryResults.find("QueryResultsRow").length).toEqual(1);
  });

  it("calls the onExportQueryResults prop when the export button is clicked", () => {
    const spy = jest.fn();
    const component = mount(
      <QueryResultsTable
        campaign={campaignWithQueryResults}
        onExportQueryResults={spy}
      />
    );

    const exportBtn = component.find(".query-results-table__export-btn");

    expect(spy).not.toHaveBeenCalled();

    exportBtn.hostNodes().simulate("click");

    expect(spy).toHaveBeenCalled();
  });
});
