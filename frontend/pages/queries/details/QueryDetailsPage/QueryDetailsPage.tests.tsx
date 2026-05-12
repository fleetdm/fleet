import React from "react";
import { screen } from "@testing-library/react";
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

const QUERY_ID = 1;
const HOST_ID = 42;

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

describe("QueryDetailsPage - back navigation", () => {
  it("'Back to host details' returns to the host's details page even when filteredQueriesPath is set in AppContext", async () => {
    setupQueryHandlers();

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          ...baseAppContext,
          // Simulates the user having visited the Queries (Reports) list earlier
          // in the session, populating filteredQueriesPath on AppContext.
          filteredQueriesPath: "/queries/manage?fleet_id=1",
        },
      },
    });

    render(<QueryDetailsPage {...(createProps(HOST_ID) as any)} />);

    const backButton = await screen.findByTestId("back-button");

    expect(backButton).toHaveTextContent("Back to host details");
    expect(backButton.getAttribute("data-path")).toContain(
      `/hosts/${HOST_ID}/details`
    );
    expect(backButton.getAttribute("data-path")).not.toBe(
      "/queries/manage?fleet_id=1"
    );
  });

  it("'Back to reports' returns to filteredQueriesPath when no hostId is present", async () => {
    setupQueryHandlers();

    const filteredPath = "/queries/manage?fleet_id=1";
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { ...baseAppContext, filteredQueriesPath: filteredPath },
      },
    });

    render(<QueryDetailsPage {...(createProps() as any)} />);

    const backButton = await screen.findByTestId("back-button");

    expect(backButton).toHaveTextContent("Back to reports");
    expect(backButton.getAttribute("data-path")).toBe(filteredPath);
  });

  it("'Back to reports' falls back to the manage reports page when neither hostId nor filteredQueriesPath is set", async () => {
    setupQueryHandlers();

    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: baseAppContext },
    });

    render(<QueryDetailsPage {...(createProps() as any)} />);

    const backButton = await screen.findByTestId("back-button");

    expect(backButton).toHaveTextContent("Back to reports");
    expect(backButton.getAttribute("data-path")).toContain("/reports/manage");
  });
});
