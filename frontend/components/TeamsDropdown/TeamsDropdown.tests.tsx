import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { noop } from "lodash";
// TODOL Replace renderWithAppContext with createCustomRenderer
import { renderWithAppContext } from "test/test-utils";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";

import TeamsDropdown from "./TeamsDropdown";

const mockPush = jest.fn();
jest.mock("react-router", () => ({
  browserHistory: {
    push: (...args: unknown[]) => mockPush(...args),
  },
}));

// The visible trigger is a real Fleet <Button>; getByRole("button") finds it
// unambiguously (react-select's hidden control has no role="button").
const getTrigger = (name: RegExp) =>
  screen.getByRole("button", { name, hidden: false });

describe("TeamsDropdown - component", () => {
  const USER_TEAMS = [
    { id: -1, name: "All fleets" },
    { id: 1, name: "Team 1" },
    { id: 2, name: "Team 2" },
  ];

  beforeEach(() => {
    mockPush.mockClear();
  });

  it("renders the given selected team from selectedTeamId", () => {
    render(
      <TeamsDropdown
        currentUserTeams={USER_TEAMS}
        selectedTeamId={1}
        onChange={noop}
      />
    );

    expect(getTrigger(/Team 1/)).toBeInTheDocument();
  });

  it("renders the first team option when includeAllTeams is false and when no selectedTeamId is given", () => {
    render(
      <TeamsDropdown
        currentUserTeams={USER_TEAMS}
        includeAllTeams={false}
        onChange={noop}
      />
    );

    expect(getTrigger(/Team 1/)).toBeInTheDocument();
  });

  describe("user is on the global team", () => {
    const contextValue = {
      isOnGlobalTeam: true,
    };

    it("renders 'All fleets' when no selectedTeamId is given", () => {
      renderWithAppContext(
        <TeamsDropdown currentUserTeams={USER_TEAMS} onChange={noop} />,
        { contextValue }
      );

      expect(getTrigger(/All fleets/)).toBeInTheDocument();
    });

    it("renders the first team option when includeAllTeams is false and when no selectedTeamId is given", () => {
      renderWithAppContext(
        <TeamsDropdown
          currentUserTeams={USER_TEAMS}
          includeAllTeams={false}
          onChange={noop}
        />,
        { contextValue }
      );

      expect(getTrigger(/Team 1/)).toBeInTheDocument();
    });
  });

  describe("user is not on the global team", () => {
    const contextValue = { isOnGlobalTeam: false };
    const filteredUserTeams = USER_TEAMS.filter(
      (t) => t.id > APP_CONTEXT_NO_TEAM_ID
    );

    it("renders the first team when no selectedTeamId is given", () => {
      renderWithAppContext(
        <TeamsDropdown currentUserTeams={filteredUserTeams} onChange={noop} />,
        { contextValue }
      );

      expect(getTrigger(/Team 1/)).toBeInTheDocument();
    });
  });

  describe("in-menu search", () => {
    // Search only appears once the list is long enough to be worth filtering.
    const MANY_TEAMS = [
      { id: -1, name: "All fleets" },
      { id: 1, name: "Team 1" },
      { id: 2, name: "Team 2" },
      { id: 3, name: "Team 3" },
      { id: 4, name: "Team 4" },
      { id: 5, name: "Team 5" },
    ];

    it("hides the search input when there are fewer than 6 fleets", async () => {
      const user = userEvent.setup();
      render(
        <TeamsDropdown
          currentUserTeams={USER_TEAMS}
          selectedTeamId={1}
          onChange={noop}
        />
      );

      await user.click(getTrigger(/Team 1/));
      expect(
        screen.queryByPlaceholderText("Search fleets")
      ).not.toBeInTheDocument();
    });

    it("renders a search input when there are 6 or more fleets", async () => {
      const user = userEvent.setup();
      render(
        <TeamsDropdown
          currentUserTeams={MANY_TEAMS}
          selectedTeamId={1}
          onChange={noop}
        />
      );

      await user.click(getTrigger(/Team 1/));
      expect(screen.getByPlaceholderText("Search fleets")).toBeInTheDocument();
    });

    it("filters options by the search query", async () => {
      const user = userEvent.setup();
      render(
        <TeamsDropdown
          currentUserTeams={MANY_TEAMS}
          selectedTeamId={1}
          onChange={noop}
        />
      );

      await user.click(getTrigger(/Team 1/));
      fireEvent.change(screen.getByPlaceholderText("Search fleets"), {
        target: { value: "Team 2" },
      });

      // The trigger button also contains "Team 1"; scope option lookups to
      // react-select's option class so the trigger doesn't count.
      const optionLabels = Array.from(
        document.querySelectorAll(".team-dropdown__option")
      ).map((o) => o.textContent);
      expect(optionLabels).toContain("Team 2");
      expect(optionLabels).not.toContain("All fleets");
    });

    it("shows the empty-state message when nothing matches", async () => {
      const user = userEvent.setup();
      render(
        <TeamsDropdown
          currentUserTeams={MANY_TEAMS}
          selectedTeamId={1}
          onChange={noop}
        />
      );

      await user.click(getTrigger(/Team 1/));
      fireEvent.change(screen.getByPlaceholderText("Search fleets"), {
        target: { value: "nothing-matches-this" },
      });

      expect(screen.getByText("No matching fleets")).toBeInTheDocument();
    });
  });

  describe("Add fleet button", () => {
    const MANY_TEAMS = [
      { id: -1, name: "All fleets" },
      { id: 1, name: "Team 1" },
      { id: 2, name: "Team 2" },
      { id: 3, name: "Team 3" },
      { id: 4, name: "Team 4" },
      { id: 5, name: "Team 5" },
    ];

    it("does not render for non-global-admin users", async () => {
      const user = userEvent.setup();
      renderWithAppContext(
        <TeamsDropdown
          currentUserTeams={USER_TEAMS}
          selectedTeamId={1}
          onChange={noop}
        />,
        { contextValue: { isGlobalAdmin: false } }
      );

      await user.click(getTrigger(/Team 1/));
      expect(
        screen.queryByRole("button", { name: /add fleet/i })
      ).not.toBeInTheDocument();
    });

    it("renders as a labeled footer for global admins when the list is short", async () => {
      const user = userEvent.setup();
      renderWithAppContext(
        <TeamsDropdown
          currentUserTeams={USER_TEAMS}
          selectedTeamId={1}
          onChange={noop}
        />,
        { contextValue: { isGlobalAdmin: true } }
      );

      await user.click(getTrigger(/Team 1/));
      const addButton = screen.getByRole("button", { name: /add fleet/i });
      expect(addButton).toHaveTextContent("Add fleet");

      await user.click(addButton);
      expect(mockPush).toHaveBeenCalledWith("/settings/fleets?create_fleet=1");
    });

    it("renders as an icon-only top button for global admins when the list is long", async () => {
      const user = userEvent.setup();
      renderWithAppContext(
        <TeamsDropdown
          currentUserTeams={MANY_TEAMS}
          selectedTeamId={1}
          onChange={noop}
        />,
        { contextValue: { isGlobalAdmin: true } }
      );

      await user.click(getTrigger(/Team 1/));
      const addButton = screen.getByRole("button", { name: /add fleet/i });
      // Icon-only variant has no visible text label.
      expect(addButton).not.toHaveTextContent("Add fleet");

      await user.click(addButton);
      expect(mockPush).toHaveBeenCalledWith("/settings/fleets?create_fleet=1");
    });
  });
});
