import React from "react";

import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";
import createMockUser from "__mocks__/userMock";

import SelectTargets from "./SelectTargets";

const MOCK_LABELS = [
  { id: 1, name: "All Hosts", label_type: "builtin", description: "" },
  { id: 2, name: "macOS", label_type: "builtin", description: "" },
];

const MOCK_TEAMS = [
  { id: 1, name: "Team Alpha", host_count: 10, user_count: 5 },
  { id: 2, name: "Team Beta", host_count: 20, user_count: 8 },
];

const labelSummariesHandler = http.get(baseUrl("/labels/summary"), () => {
  return HttpResponse.json({ labels: MOCK_LABELS });
});

const teamsHandler = http.get(baseUrl("/fleets"), () => {
  return HttpResponse.json({ teams: MOCK_TEAMS });
});

const defaultProps = {
  baseClass: "select-targets",
  selectedTargets: [],
  targetedHosts: [],
  targetedLabels: [],
  targetedTeams: [],
  goToQueryEditor: jest.fn(),
  goToRunQuery: jest.fn(),
  setSelectedTargets: jest.fn(),
  setTargetedHosts: jest.fn(),
  setTargetedLabels: jest.fn(),
  setTargetedTeams: jest.fn(),
  setTargetsTotalCount: jest.fn(),
};

const getTeamButton = (name: string) =>
  screen.getByText(name).closest("button");

describe("SelectTargets - team disabling", () => {
  beforeEach(() => {
    mockServer.use(labelSummariesHandler, teamsHandler);
  });

  describe("plain observer (not observer+)", () => {
    const plainObserverOnBothTeams = createMockUser({
      global_role: null,
      teams: [
        { ...MOCK_TEAMS[0], role: "observer" },
        { ...MOCK_TEAMS[1], role: "observer" },
      ],
    });

    it("disables all observer teams for live policies", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: plainObserverOnBothTeams,
            isPremiumTier: true,
            isOnGlobalTeam: false,
          },
        },
      });

      render(
        <SelectTargets
          {...defaultProps}
          isLivePolicy
          isObserverCanRunQuery={false}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Team Alpha")).toBeDisabled();
        expect(getTeamButton("Team Beta")).toBeDisabled();
      });
    });

    it("disables observer teams when query does not have observer_can_run", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: plainObserverOnBothTeams,
            isPremiumTier: true,
            isOnGlobalTeam: false,
          },
        },
      });

      render(
        <SelectTargets
          {...defaultProps}
          isObserverCanRunQuery={false}
          queryTeamId={1}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Team Alpha")).toBeDisabled();
        expect(getTeamButton("Team Beta")).toBeDisabled();
      });
    });

    it("enables only the query's team when observer_can_run is true", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: plainObserverOnBothTeams,
            isPremiumTier: true,
            isOnGlobalTeam: false,
          },
        },
      });

      // Query belongs to team 1; observer may only target that team.
      render(
        <SelectTargets
          {...defaultProps}
          isObserverCanRunQuery
          queryTeamId={1}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Team Alpha")).toBeEnabled();
        expect(getTeamButton("Team Beta")).toBeDisabled();
      });
    });

    it("enables only the query's team when observer_can_run is true (query on team 2)", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: plainObserverOnBothTeams,
            isPremiumTier: true,
            isOnGlobalTeam: false,
          },
        },
      });

      // Query belongs to team 2; only team 2 should be enabled.
      render(
        <SelectTargets
          {...defaultProps}
          isObserverCanRunQuery
          queryTeamId={2}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Team Alpha")).toBeDisabled();
        expect(getTeamButton("Team Beta")).toBeEnabled();
      });
    });
  });

  describe("observer+ user", () => {
    const observerPlusOnBothTeams = createMockUser({
      global_role: null,
      teams: [
        { ...MOCK_TEAMS[0], role: "observer_plus" },
        { ...MOCK_TEAMS[1], role: "observer_plus" },
      ],
    });

    it("enables all teams for observer+ even on live policies", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: observerPlusOnBothTeams,
            isPremiumTier: true,
            isOnGlobalTeam: false,
          },
        },
      });

      render(
        <SelectTargets
          {...defaultProps}
          isLivePolicy
          isObserverCanRunQuery={false}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Team Alpha")).toBeEnabled();
        expect(getTeamButton("Team Beta")).toBeEnabled();
      });
    });

    it("enables all teams for observer+ on queries", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: observerPlusOnBothTeams,
            isPremiumTier: true,
            isOnGlobalTeam: false,
          },
        },
      });

      render(
        <SelectTargets
          {...defaultProps}
          isObserverCanRunQuery={false}
          queryTeamId={1}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Team Alpha")).toBeEnabled();
        expect(getTeamButton("Team Beta")).toBeEnabled();
      });
    });
  });

  describe("multi-team user with mixed roles (admin on team 1, observer on team 2)", () => {
    const adminObsUser = createMockUser({
      global_role: null,
      teams: [
        { ...MOCK_TEAMS[0], role: "admin" },
        { ...MOCK_TEAMS[1], role: "observer" },
      ],
    });

    it("disables only the observer team for live policies", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: adminObsUser,
            isPremiumTier: true,
            isOnGlobalTeam: false,
          },
        },
      });

      render(
        <SelectTargets
          {...defaultProps}
          isLivePolicy
          isObserverCanRunQuery={false}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Team Alpha")).toBeEnabled();
        expect(getTeamButton("Team Beta")).toBeDisabled();
      });
    });

    it("disables observer team when observer_can_run query belongs to a different team", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: adminObsUser,
            isPremiumTier: true,
            isOnGlobalTeam: false,
          },
        },
      });

      // Query belongs to team 1 (admin team), observer_can_run is true.
      // Team 2 (observer role) should be disabled — observer_can_run is scoped to the query's team.
      render(
        <SelectTargets
          {...defaultProps}
          isObserverCanRunQuery
          queryTeamId={1}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Team Alpha")).toBeEnabled();
        expect(getTeamButton("Team Beta")).toBeDisabled();
      });
    });

    it("enables observer team when observer_can_run query belongs to that team", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: adminObsUser,
            isPremiumTier: true,
            isOnGlobalTeam: false,
          },
        },
      });

      // Query belongs to team 2 (observer team) and observer_can_run is true
      render(
        <SelectTargets
          {...defaultProps}
          isObserverCanRunQuery
          queryTeamId={2}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Team Alpha")).toBeEnabled();
        expect(getTeamButton("Team Beta")).toBeEnabled();
      });
    });
  });

  describe("global observer", () => {
    const globalObserver = createMockUser({
      global_role: "observer",
      teams: [],
    });

    it("disables all teams (including Unassigned) for live policies", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: globalObserver,
            isPremiumTier: true,
            isOnGlobalTeam: true,
          },
        },
      });

      render(
        <SelectTargets
          {...defaultProps}
          isLivePolicy
          isObserverCanRunQuery={false}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Unassigned")).toBeDisabled();
        expect(getTeamButton("Team Alpha")).toBeDisabled();
        expect(getTeamButton("Team Beta")).toBeDisabled();
      });
    });

    it("disables all teams when query does not have observer_can_run", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: globalObserver,
            isPremiumTier: true,
            isOnGlobalTeam: true,
          },
        },
      });

      render(
        <SelectTargets
          {...defaultProps}
          isObserverCanRunQuery={false}
          queryTeamId={1}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Unassigned")).toBeDisabled();
        expect(getTeamButton("Team Alpha")).toBeDisabled();
        expect(getTeamButton("Team Beta")).toBeDisabled();
      });
    });

    it("enables only the query's team when observer_can_run is true", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: globalObserver,
            isPremiumTier: true,
            isOnGlobalTeam: true,
          },
        },
      });

      // Query belongs to team 1; global observer may only target that team.
      render(
        <SelectTargets
          {...defaultProps}
          isObserverCanRunQuery
          queryTeamId={1}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Unassigned")).toBeDisabled();
        expect(getTeamButton("Team Alpha")).toBeEnabled();
        expect(getTeamButton("Team Beta")).toBeDisabled();
      });
    });

    it("enables all teams for a global observer_can_run query (no team_id)", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: globalObserver,
            isPremiumTier: true,
            isOnGlobalTeam: true,
          },
        },
      });

      render(
        <SelectTargets
          {...defaultProps}
          isObserverCanRunQuery
          queryTeamId={null}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Unassigned")).toBeEnabled();
        expect(getTeamButton("Team Alpha")).toBeEnabled();
        expect(getTeamButton("Team Beta")).toBeEnabled();
      });
    });
  });

  describe("helper text visibility", () => {
    it("shows helper text when some fleets are disabled", async () => {
      const plainObserver = createMockUser({
        global_role: "observer",
        teams: [],
      });

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: plainObserver,
            isPremiumTier: true,
            isOnGlobalTeam: true,
          },
        },
      });

      render(
        <SelectTargets
          {...defaultProps}
          isObserverCanRunQuery={false}
          queryTeamId={1}
        />
      );

      await waitFor(() => {
        expect(
          screen.getByText("Results limited to fleets you can access.")
        ).toBeInTheDocument();
      });
    });

    it("does not show helper text when no fleets are disabled", async () => {
      const globalAdmin = createMockUser({
        global_role: "admin",
        teams: [],
      });

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: globalAdmin,
            isPremiumTier: true,
            isOnGlobalTeam: true,
          },
        },
      });

      render(
        <SelectTargets
          {...defaultProps}
          isObserverCanRunQuery={false}
          queryTeamId={1}
        />
      );

      // Wait for teams to render to confirm loading is done before asserting absence
      await waitFor(() => {
        expect(getTeamButton("Team Alpha")).toBeInTheDocument();
      });
      expect(
        screen.queryByText("Results limited to fleets you can access.")
      ).not.toBeInTheDocument();
    });
  });

  describe("global observer+", () => {
    const globalObserverPlus = createMockUser({
      global_role: "observer_plus",
      teams: [],
    });

    it("enables all teams for global observer+ even on live policies", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: globalObserverPlus,
            isPremiumTier: true,
            isOnGlobalTeam: true,
          },
        },
      });

      render(
        <SelectTargets
          {...defaultProps}
          isLivePolicy
          isObserverCanRunQuery={false}
        />
      );

      await waitFor(() => {
        expect(getTeamButton("Unassigned")).toBeEnabled();
        expect(getTeamButton("Team Alpha")).toBeEnabled();
        expect(getTeamButton("Team Beta")).toBeEnabled();
      });
    });
  });
});
