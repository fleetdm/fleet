import React from "react";
import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import { createCustomRenderer, createMockRouter, baseUrl } from "test/test-utils";
import mockServer from "test/mock-server";
import createMockUser from "__mocks__/userMock";

import HostReportsTab from "./HostReportsTab";

const mockRouter = createMockRouter();

const emptyReportsResponse = {
  reports: [],
  count: 0,
  meta: { has_previous_results: false, has_next_results: false },
};

const populatedReportsResponse = {
  reports: [
    {
      report_id: 1,
      name: "Test Report",
      description: "A test report",
      last_fetched: "2024-01-01T00:00:00Z",
      first_result: { col1: "val1" },
      n_host_results: 5,
      report_clipped: false,
      store_results: true,
    },
  ],
  count: 1,
  meta: { has_previous_results: false, has_next_results: false },
};

const createHostReportsHandler = (response: Record<string, unknown>) =>
  http.get(baseUrl("/hosts/:id/reports"), () => {
    return HttpResponse.json(response);
  });

const baseProps = {
  hostId: 1,
  hostName: "test-host",
  router: mockRouter,
  location: {
    pathname: "/hosts/1",
    query: {},
  },
};

describe("HostReportsTab", () => {
  it("renders truly empty state with disabled controls and Schedule a report CTA", async () => {
    mockServer.use(createHostReportsHandler(emptyReportsResponse));
    const onScheduleReport = jest.fn();
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <HostReportsTab
        {...baseProps}
        canScheduleReport
        onScheduleReport={onScheduleReport}
      />
    );

    // Empty state copy
    expect(
      await screen.findByText("No reports scheduled")
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /Select Refetch to load the latest data from this host, or schedule a report/
      )
    ).toBeInTheDocument();

    // Schedule a report CTA
    expect(
      screen.getByRole("button", { name: /schedule a report/i })
    ).toBeInTheDocument();

    // Shows 0 reports count
    expect(screen.getByText("0 reports")).toBeInTheDocument();

    // Search is disabled
    const searchInput = screen.getByPlaceholderText(/search by name/i);
    expect(searchInput).toBeDisabled();
  });

  it("renders truly empty state without Schedule CTA when user lacks permission", async () => {
    mockServer.use(createHostReportsHandler(emptyReportsResponse));
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isOnlyObserver: true,
          currentUser: createMockUser({ global_role: "observer" }),
        },
      },
    });

    render(
      <HostReportsTab {...baseProps} canScheduleReport={false} />
    );

    expect(
      await screen.findByText("No reports scheduled")
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /schedule a report/i })
    ).not.toBeInTheDocument();
  });

  it("renders populated state with enabled controls", async () => {
    mockServer.use(createHostReportsHandler(populatedReportsResponse));
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(<HostReportsTab {...baseProps} />);

    expect(await screen.findByText("1 report")).toBeInTheDocument();

    // Search is NOT disabled
    const searchInput = screen.getByPlaceholderText(/search by name/i);
    expect(searchInput).not.toBeDisabled();
  });
});
