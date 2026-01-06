import React from "react";
import { screen, render } from "@testing-library/react";
import { noop } from "lodash";

import { createMockRouter } from "test/test-utils";
import { IQueryStats } from "interfaces/query_stats";

import HostQueries from ".";

const createMockQueryStats = (
  overrides?: Partial<IQueryStats>
): IQueryStats => ({
  scheduled_query_name: "test-query",
  scheduled_query_id: 1,
  query_name: "Test Query",
  discard_data: false,
  last_fetched: "2025-01-01T00:00:00Z",
  automations_enabled: false,
  description: "A test query",
  pack_name: "test-pack",
  pack_id: 1,
  average_memory: 100,
  denylisted: false,
  executions: 10,
  interval: 3600,
  last_executed: "2025-01-01T00:00:00Z",
  output_size: 1024,
  system_time: 50,
  user_time: 100,
  wall_time: 150,
  ...overrides,
});

describe("HostQueries card", () => {
  it("renders the queries table and add query button for supported platform with queries", () => {
    const schedule = [
      createMockQueryStats({ query_name: "Query 1", scheduled_query_id: 1 }),
      createMockQueryStats({ query_name: "Query 2", scheduled_query_id: 2 }),
    ];

    render(
      <HostQueries
        hostId={1}
        schedule={schedule}
        hostPlatform="darwin"
        router={createMockRouter()}
        canAddQuery
        onClickAddQuery={noop}
      />
    );

    expect(screen.getByText("Queries")).toBeInTheDocument();
    expect(screen.getByText("Add query")).toBeInTheDocument();
    // Use getAllByText due to tooltip duplicates
    expect(screen.getAllByText("Query 1").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Query 2").length).toBeGreaterThan(0);
  });

  it("renders 'Queries not supported for this host' message and hides add query button for unsupported host platform", () => {
    render(
      <HostQueries
        hostId={1}
        schedule={[]}
        hostPlatform="chrome"
        router={createMockRouter()}
        canAddQuery={false}
        onClickAddQuery={noop}
      />
    );

    expect(screen.getByText("Queries")).toBeInTheDocument();
    expect(
      screen.getByText("Queries not supported for this host")
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Interested in collecting data from your Chromebooks/)
    ).toBeInTheDocument();
    expect(screen.queryByText("Add query")).not.toBeInTheDocument();
  });

  it("renders empty state and add query button for supported platform with no queries", () => {
    render(
      <HostQueries
        hostId={1}
        schedule={[]}
        hostPlatform="darwin"
        router={createMockRouter()}
        canAddQuery
        onClickAddQuery={noop}
      />
    );

    expect(screen.getByText("Queries")).toBeInTheDocument();
    expect(screen.getByText("Add query")).toBeInTheDocument();
    expect(screen.getByText("No queries")).toBeInTheDocument();
    expect(
      screen.getByText("Add a query to view custom vitals.")
    ).toBeInTheDocument();
  });
});
