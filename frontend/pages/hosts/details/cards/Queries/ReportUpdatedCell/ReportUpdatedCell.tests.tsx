import React from "react";

import { render, screen } from "@testing-library/react";

import ReportUpdatedCell from "./ReportUpdatedCell";

describe("ReportUpdatedCell component", () => {
  it("Renders '---' with tooltip and no link when run on an interval with discard data and automations enabled", () => {
    render(
      <ReportUpdatedCell
        interval={1000}
        discard_data
        automations_enabled
        should_link_to_hqr={false}
      />
    );

    expect(screen.getByText(/---/)).toBeInTheDocument();
    expect(screen.getByText(/Results from this query/)).toBeInTheDocument();
    expect(screen.queryByText(/View report/)).toBeNull();
  });

  it("Renders 'Never with tooltip and link to report when run on an interval with discard data off and no last_fetched time", () => {
    render(
      <ReportUpdatedCell
        interval={1000}
        discard_data={false}
        automations_enabled={false}
        should_link_to_hqr
      />
    );

    expect(screen.getByText(/Never/)).toBeInTheDocument();
    expect(screen.getByText(/This query has not run/)).toBeInTheDocument();
    expect(screen.getByText(/View report/)).toBeInTheDocument();
  });

  it("Renders a last-updated timestamp with tooltip and link to report when a last_fetched date is present", () => {
    const tenDaysAgo = new Date();
    tenDaysAgo.setDate(tenDaysAgo.getDate() - 10);
    render(
      <ReportUpdatedCell
        interval={1000}
        discard_data={false}
        automations_enabled={false}
        should_link_to_hqr
        last_fetched={tenDaysAgo.toISOString()}
      />
    );

    expect(
      screen.getByText(/\d\d\/\d\d\/\d\d\d\d, \d{1,2}:\d{1,2}:\d{1,2}( AM|PM)?/)
    ).toBeInTheDocument();
    expect(screen.getByText(/\d+.+ago/)).toBeInTheDocument();
    expect(screen.getByText(/View report/)).toBeInTheDocument();
  });
  it("Renders a last-updated timestamp with tooltip and link to report when a last_fetched date is present but not currently running an interval", () => {
    const tenDaysAgo = new Date();
    tenDaysAgo.setDate(tenDaysAgo.getDate() - 10);
    render(
      <ReportUpdatedCell
        interval={0}
        discard_data={false}
        automations_enabled={false}
        should_link_to_hqr
        last_fetched={tenDaysAgo.toISOString()}
      />
    );

    expect(
      screen.getByText(/\d\d\/\d\d\/\d\d\d\d, \d{1,2}:\d{1,2}:\d{1,2}( AM|PM)?/)
    ).toBeInTheDocument();
    expect(screen.getByText(/\d+.+ago/)).toBeInTheDocument();
    expect(screen.getByText(/View report/)).toBeInTheDocument();
  });
});
