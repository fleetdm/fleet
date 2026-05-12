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
