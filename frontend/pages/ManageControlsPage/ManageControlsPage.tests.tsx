import React from "react";
import { screen } from "@testing-library/react";

import PATHS from "router/paths";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import ManageControlsPage from "./ManageControlsPage";

// Drive teamIdForApi directly so we can assert how ManageControlsPage forwards
// it to its sub-pages. On free tier useTeamIdParam yields `undefined` (Controls
// runs as "All teams", which it doesn't support); the page must coerce that to
// "No team" (0) so sub-pages don't spin forever.
const mockUseTeamIdParam = jest.fn();
jest.mock("hooks/useTeamIdParam", () => ({
  __esModule: true,
  default: () => mockUseTeamIdParam(),
}));

// A stand-in Controls sub-page that surfaces the teamIdForApi it's handed.
const TestSubPage = ({ teamIdForApi }: { teamIdForApi?: number }) => (
  <div data-testid="sub-page">teamIdForApi={String(teamIdForApi)}</div>
);

const renderPage = (
  isPremiumTier: boolean,
  teamIdForApi: number | undefined
) => {
  mockUseTeamIdParam.mockReturnValue({
    currentTeamId: teamIdForApi,
    userTeams: [],
    teamIdForApi,
    handleTeamChange: jest.fn(),
  });

  const render = createCustomRenderer({
    withBackendMock: true,
    context: { app: { isPremiumTier, isGlobalAdmin: true } },
  });

  return render(
    <ManageControlsPage
      location={{
        pathname: PATHS.CONTROLS_OS_SETTINGS,
        search: "",
        query: {},
      }}
      router={createMockRouter()}
    >
      <TestSubPage />
    </ManageControlsPage>
  );
};

describe("ManageControlsPage - teamIdForApi forwarded to sub-pages", () => {
  afterEach(() => jest.clearAllMocks());

  it("coerces undefined teamIdForApi to 'No team' (0) on free tier so sub-pages don't spin forever", () => {
    renderPage(false, undefined);
    expect(screen.getByTestId("sub-page")).toHaveTextContent("teamIdForApi=0");
  });

  it("keeps teamIdForApi undefined on premium while the selected fleet resolves (preserves the loading gate)", () => {
    renderPage(true, undefined);
    expect(screen.getByTestId("sub-page")).toHaveTextContent(
      "teamIdForApi=undefined"
    );
  });

  it("forwards the resolved teamIdForApi on premium", () => {
    renderPage(true, 5);
    expect(screen.getByTestId("sub-page")).toHaveTextContent("teamIdForApi=5");
  });
});
