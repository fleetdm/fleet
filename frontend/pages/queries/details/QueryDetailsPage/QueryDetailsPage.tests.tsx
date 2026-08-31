import React from "react";
import { screen, waitFor, act } from "@testing-library/react";
import { focusManager } from "react-query";
import { http, HttpResponse } from "msw";

import {
  createCustomRenderer,
  baseUrl,
  createMockRouter,
} from "test/test-utils";
import mockServer from "test/mock-server";
import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";
import createMockSchedulableQuery from "__mocks__/scheduleableQueryMock";
import createMockQueryReport from "__mocks__/queryReportMock";
import { IQueryReportResultRow } from "interfaces/query_report";

import QueryDetailsPage from "./QueryDetailsPage";

// Render BackButton as a button exposing its `path` prop so we can assert on the
// computed back URL without depending on react-router's browserHistory.
jest.mock("components/BackButton", () => ({
  __esModule: true,
  default: ({ text, path }: { text: string; path?: string }) => (
    <button type="button" data-testid="back-button" data-path={path ?? ""}>
      {text}
    </button>
  ),
}));

// Surface the modal's `query` prop as plain text (the real modal renders it in
// an Ace editor that isn't reliably assertable in jsdom).
jest.mock("components/modals/ShowQueryModal", () => ({
  __esModule: true,
  default: ({ query }: { query?: string }) => (
    <div data-testid="show-query-modal">{query}</div>
  ),
}));

// Surface the report-setting props derived from the loaded query so we can
// assert they reflect storedQuery rather than stale context.
jest.mock("../components/NoResults/NoResults", () => ({
  __esModule: true,
  default: ({
    discardDataEnabled,
    loggingSnapshot,
  }: {
    discardDataEnabled: boolean;
    loggingSnapshot: boolean;
  }) => (
    <div
      data-testid="no-results"
      data-discard={String(discardDataEnabled)}
      data-snapshot={String(loggingSnapshot)}
    />
  ),
}));

const QUERY_ID = 1;
const HOST_ID = 42;
const FILTERED_QUERIES_PATH = "/queries/manage?fleet_id=1";

const setupQueryHandlers = () => {
  mockServer.use(
    http.get(baseUrl(`/reports/${QUERY_ID}`), () =>
      HttpResponse.json({
        query: createMockSchedulableQuery({ id: QUERY_ID, team_id: null }),
      })
    ),
    http.get(baseUrl(`/reports/${QUERY_ID}/report`), () =>
      HttpResponse.json(
        createMockQueryReport({ query_id: QUERY_ID, results: [] })
      )
    )
  );
};

const createProps = (hostId?: number) => ({
  router: createMockRouter(),
  params: { id: String(QUERY_ID) },
  location: {
    pathname: `/reports/${QUERY_ID}`,
    search: hostId !== undefined ? `?host_id=${hostId}` : "",
    query: hostId !== undefined ? { host_id: String(hostId) } : {},
  },
});

const baseAppContext = {
  isGlobalAdmin: true,
  isOnGlobalTeam: true,
  // Free tier short-circuits useTeamIdParam's redirect logic when no fleet_id is set,
  // keeping the test focused on backPath() rather than team reconciliation.
  isFreeTier: true,
  isPremiumTier: false,
  currentUser: createMockUser({ global_role: "admin" }),
  config: createMockConfig(),
  availableTeams: [],
};

const renderPage = (
  appOverrides: { filteredQueriesPath?: string } = {},
  hostId?: number
) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: { app: { ...baseAppContext, ...appOverrides } },
  });
  render(<QueryDetailsPage {...(createProps(hostId) as any)} />);
  return screen.findByTestId("back-button");
};

describe("QueryDetailsPage - renders fresh query data (regression #43310)", () => {
  it("renders the loaded query's fields, not stale QueryContext values", async () => {
    mockServer.use(
      http.get(baseUrl(`/reports/${QUERY_ID}`), () =>
        HttpResponse.json({
          query: createMockSchedulableQuery({
            id: QUERY_ID,
            team_id: null,
            name: "Fresh report name",
            description: "Fresh report description",
            query: "SELECT 'fresh';",
            logging: "differential", // not "snapshot"
            discard_data: true,
          }),
        })
      ),
      http.get(baseUrl(`/reports/${QUERY_ID}/report`), () =>
        HttpResponse.json(
          createMockQueryReport({ query_id: QUERY_ID, results: [] })
        )
      )
    );

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: baseAppContext,
        // Stale values left over from a previously-viewed report. The page must
        // ignore all of these and render the freshly-loaded query instead.
        query: {
          lastEditedQueryName: "Stale report name",
          lastEditedQueryDescription: "Stale report description",
          lastEditedQueryBody: "SELECT 'stale';",
          lastEditedQueryLoggingType: "snapshot",
          lastEditedQueryDiscardData: false,
        },
      },
    });
    const { user } = render(<QueryDetailsPage {...(createProps() as any)} />);

    // name + description
    expect(await screen.findByText("Fresh report name")).toBeInTheDocument();
    expect(screen.getByText("Fresh report description")).toBeInTheDocument();
    expect(screen.queryByText("Stale report name")).not.toBeInTheDocument();
    expect(
      screen.queryByText("Stale report description")
    ).not.toBeInTheDocument();

    // logging + discard_data (drive the report's caching state)
    const noResults = screen.getByTestId("no-results");
    expect(noResults).toHaveAttribute("data-discard", "true");
    expect(noResults).toHaveAttribute("data-snapshot", "false");

    // query (shown via the "Show query" modal)
    await user.click(screen.getByRole("button", { name: "Show query" }));
    expect(screen.getByTestId("show-query-modal")).toHaveTextContent(
      "SELECT 'fresh';"
    );
    expect(screen.getByTestId("show-query-modal")).not.toHaveTextContent(
      "SELECT 'stale';"
    );
  });

  it("derives Live report visibility from the loaded query's observer_can_run, not stale context", async () => {
    mockServer.use(
      http.get(baseUrl(`/reports/${QUERY_ID}`), () =>
        HttpResponse.json({
          query: createMockSchedulableQuery({
            id: QUERY_ID,
            team_id: null,
            observer_can_run: true,
          }),
        })
      ),
      http.get(baseUrl(`/reports/${QUERY_ID}/report`), () =>
        HttpResponse.json(
          createMockQueryReport({ query_id: QUERY_ID, results: [] })
        )
      )
    );

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        // A plain observer: the only thing that can grant live-query access here
        // is the query's own observer_can_run, so the button proves the source.
        app: {
          ...baseAppContext,
          isGlobalAdmin: false,
          isGlobalMaintainer: false,
          isObserverPlus: false,
          isGlobalTechnician: false,
          isTeamMaintainerOrTeamAdmin: false,
          isTeamTechnician: false,
          isOnGlobalTeam: false,
          currentUser: createMockUser({ global_role: "observer" }),
        },
        query: { lastEditedQueryObserverCanRun: false },
      },
    });
    render(<QueryDetailsPage {...(createProps() as any)} />);

    expect(
      await screen.findByRole("button", { name: /Live report/i })
    ).toBeInTheDocument();
  });
});

describe("QueryDetailsPage - back navigation", () => {
  beforeEach(() => setupQueryHandlers());

  it.each([
    {
      name: "hostId wins over filteredQueriesPath",
      app: { filteredQueriesPath: FILTERED_QUERIES_PATH },
      hostId: HOST_ID,
      expectText: "Back to host details",
      expectPathContains: `/hosts/${HOST_ID}/details`,
    },
    {
      name: "falls back to filteredQueriesPath when no hostId",
      app: { filteredQueriesPath: FILTERED_QUERIES_PATH },
      hostId: undefined,
      expectText: "Back to reports",
      expectPathContains: FILTERED_QUERIES_PATH,
    },
    {
      name:
        "falls back to manage reports when neither hostId nor filteredQueriesPath is set",
      app: {},
      hostId: undefined,
      expectText: "Back to reports",
      expectPathContains: "/reports/manage",
    },
  ])("$name", async ({ app, hostId, expectText, expectPathContains }) => {
    const back = await renderPage(app, hostId);
    expect(back).toHaveTextContent(expectText);
    expect(back.getAttribute("data-path")).toContain(expectPathContains);
  });
});

const RESULT_ROW: IQueryReportResultRow = {
  host_id: 1,
  host_name: "alpha-host",
  last_fetched: "2024-01-01T00:00:00Z",
  columns: { model: "WIDGET-XYZ" },
};

// Sets up the metadata + report handlers, with the report results determined by
// `reportSequence(callIndex)` so a test can return different results per fetch
// (e.g. empty first, then populated).
const setupReportHandlers = ({
  queryOverrides = {},
  reportSequence,
}: {
  queryOverrides?: Parameters<typeof createMockSchedulableQuery>[0];
  reportSequence: (callIndex: number) => IQueryReportResultRow[];
}) => {
  let reportCalls = 0;
  let metadataCalls = 0;
  mockServer.use(
    http.get(baseUrl(`/reports/${QUERY_ID}`), () => {
      metadataCalls += 1;
      return HttpResponse.json({
        query: createMockSchedulableQuery({
          id: QUERY_ID,
          team_id: null,
          ...queryOverrides,
        }),
      });
    }),
    http.get(baseUrl(`/reports/${QUERY_ID}/report`), () => {
      const results = reportSequence(reportCalls);
      reportCalls += 1;
      return HttpResponse.json(
        createMockQueryReport({ query_id: QUERY_ID, results })
      );
    })
  );
  return {
    getReportCalls: () => reportCalls,
    getMetadataCalls: () => metadataCalls,
  };
};

const renderReportPage = () => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: { app: baseAppContext },
  });
  return render(<QueryDetailsPage {...createProps()} />);
};

describe("QueryDetailsPage - report results states", () => {
  it("shows the no-results empty state when the report has no results", async () => {
    setupReportHandlers({ reportSequence: () => [] });
    renderReportPage();

    expect(await screen.findByTestId("no-results")).toBeInTheDocument();
    expect(screen.queryByText("WIDGET-XYZ")).not.toBeInTheDocument();
  });

  it("shows the results table when the report has results", async () => {
    setupReportHandlers({ reportSequence: () => [RESULT_ROW] });
    renderReportPage();

    expect(await screen.findByText("WIDGET-XYZ")).toBeInTheDocument();
    expect(screen.queryByTestId("no-results")).not.toBeInTheDocument();
  });
});

describe("QueryDetailsPage - report refetching", () => {
  afterEach(() => {
    focusManager.setFocused(undefined);
  });

  it("shows results after a refetch when a previously-empty report returns rows", async () => {
    // Empty on first load, populated on the second fetch.
    setupReportHandlers({
      reportSequence: (callIndex) => (callIndex === 0 ? [] : [RESULT_ROW]),
    });
    renderReportPage();

    // Initially empty.
    expect(await screen.findByTestId("no-results")).toBeInTheDocument();

    // Trigger a window focus refetch.
    act(() => {
      focusManager.setFocused(true);
    });

    expect(await screen.findByText("WIDGET-XYZ")).toBeInTheDocument();
    expect(screen.queryByTestId("no-results")).not.toBeInTheDocument();
  });

  it("does not refetch the report when caching is disabled (discard_data = true)", async () => {
    // With caching disabled the report can never populate, so we must not keep
    // hitting the report endpoint even though the next fetch would return rows.
    const handlers = setupReportHandlers({
      queryOverrides: { discard_data: true },
      reportSequence: (callIndex) => (callIndex === 0 ? [] : [RESULT_ROW]),
    });
    renderReportPage();

    expect(await screen.findByTestId("no-results")).toBeInTheDocument();
    expect(handlers.getReportCalls()).toBe(1);

    act(() => {
      focusManager.setFocused(true);
    });

    await waitFor(() =>
      expect(handlers.getMetadataCalls()).toBeGreaterThanOrEqual(2)
    );
    expect(handlers.getReportCalls()).toBe(1);
    expect(screen.getByTestId("no-results")).toBeInTheDocument();
  });
});
