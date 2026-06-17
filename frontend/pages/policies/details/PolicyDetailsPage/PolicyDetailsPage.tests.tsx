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
import createMockPolicy from "__mocks__/policyMock";

import PolicyDetailsPage from "./PolicyDetailsPage";

// Avoid depending on react-router's browserHistory inside BackButton.
jest.mock("components/BackButton", () => ({
  __esModule: true,
  default: ({ text }: { text: string }) => (
    <button type="button" data-testid="back-button">
      {text}
    </button>
  ),
}));

const POLICY_ID = 8;

const createProps = () => ({
  router: createMockRouter(),
  params: { id: String(POLICY_ID) },
  location: {
    pathname: `/policies/${POLICY_ID}`,
    search: "",
    query: {},
  },
});

const baseAppContext = {
  isGlobalAdmin: true,
  isOnGlobalTeam: true,
  // Free tier short-circuits useTeamIdParam's redirect logic when no fleet_id is
  // set, keeping the test focused on which data source the page renders from.
  isFreeTier: true,
  isPremiumTier: false,
  currentUser: createMockUser({ global_role: "admin" }),
  config: createMockConfig(),
  availableTeams: [],
};

describe("PolicyDetailsPage - renders fresh policy data (regression #43310)", () => {
  it("shows the loaded policy's name/description, not stale PolicyContext values", async () => {
    mockServer.use(
      // team_id: null keeps the team query disabled, so no second endpoint to mock.
      http.get(baseUrl(`/policies/${POLICY_ID}`), () =>
        HttpResponse.json({
          policy: createMockPolicy({
            id: POLICY_ID,
            team_id: null,
            name: "Fresh policy name",
            description: "Fresh policy description",
          }),
        })
      )
    );

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: baseAppContext,
        // Stale values left over from a previously-viewed policy. The page must
        // ignore these and render the freshly-loaded policy instead.
        policy: {
          lastEditedQueryName: "Stale policy name",
          lastEditedQueryDescription: "Stale policy description",
        },
      },
    });
    render(<PolicyDetailsPage {...(createProps() as any)} />);

    expect(await screen.findByText("Fresh policy name")).toBeInTheDocument();
    expect(screen.getByText("Fresh policy description")).toBeInTheDocument();
    expect(screen.queryByText("Stale policy name")).not.toBeInTheDocument();
    expect(
      screen.queryByText("Stale policy description")
    ).not.toBeInTheDocument();
  });
});
