import React from "react";

import { screen, waitFor } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import ReportUpdatedCell from "./ReportUpdatedCell";

const HUMAN_READABLE_DATETIME_REGEX = /\d{1,2}\/\d{1,2}\/\d\d\d\d, \d{1,2}:\d{1,2}:\d{1,2}\s(A|P)M/;

describe("ReportUpdatedCell component", () => {
  it("Renders 'No report' with tooltip and no link when run on an interval with discard data and automations enabled", async () => {
    const { user } = renderWithSetup(
      <ReportUpdatedCell
        interval={1000}
        discard_data
        automations_enabled
        should_link_to_hqr={false}
        queryId={3}
        hostId={4}
      />
    );
    const noReportText = screen.getByText(/No report/);
    expect(noReportText).toBeInTheDocument();
    await user.hover(noReportText);

    waitFor(() => {
      expect(screen.getByText(/Results from this report/)).toBeInTheDocument();
    });
    expect(screen.queryByText(/View report/)).toBeNull();
  });

  it("Renders '---' with tooltip and link to report when run on an interval with discard data off and no last_fetched time", async () => {
    const { user } = renderWithSetup(
      <ReportUpdatedCell
        interval={1000}
        discard_data={false}
        automations_enabled={false}
        should_link_to_hqr
        queryId={3}
        hostId={4}
      />
    );

    const noReportText = screen.getByText(/---/);
    expect(noReportText).toBeInTheDocument();
    await user.hover(noReportText);

    waitFor(() => {
      expect(
        screen.getByText(/Fleet is collecting report results\./)
      ).toBeInTheDocument();
      expect(screen.getByText(/Check back later./)).toBeInTheDocument();
    });
  });

  it("Renders a last-updated timestamp with tooltip and link to report when a last_fetched date is present", async () => {
    const tenDaysAgo = new Date();
    tenDaysAgo.setDate(tenDaysAgo.getDate() - 10);
    const { user } = renderWithSetup(
      <ReportUpdatedCell
        interval={1000}
        discard_data={false}
        automations_enabled={false}
        should_link_to_hqr
        last_fetched={tenDaysAgo.toISOString()}
        queryId={3}
        hostId={4}
      />
    );
    const timeAgo = screen.getByText(/\d+.+ago/);

    expect(timeAgo).toBeInTheDocument();
    await user.hover(timeAgo);

    await waitFor(() => {
      expect(
        screen.getByText(HUMAN_READABLE_DATETIME_REGEX)
      ).toBeInTheDocument();
    });
    expect(screen.getByText(/View data/)).toBeInTheDocument();
  });
  it("Renders a last-updated timestamp with tooltip and link to report when a last_fetched date is present but not currently running an interval", async () => {
    const tenDaysAgo = new Date();
    tenDaysAgo.setDate(tenDaysAgo.getDate() - 10);
    const { user } = renderWithSetup(
      <ReportUpdatedCell
        interval={0}
        discard_data={false}
        automations_enabled={false}
        should_link_to_hqr
        last_fetched={tenDaysAgo.toISOString()}
        queryId={3}
        hostId={4}
      />
    );
    const timeAgo = screen.getByText(/\d+.+ago/);
    expect(timeAgo).toBeInTheDocument();
    await user.hover(timeAgo);

    await waitFor(() => {
      expect(
        screen.getByText(HUMAN_READABLE_DATETIME_REGEX)
      ).toBeInTheDocument();
    });
    expect(screen.getByText(/View data/)).toBeInTheDocument();
  });
});
