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

    const selectedTeam = screen.getByText("Team 1");
    expect(selectedTeam).toBeInTheDocument();
  });

  it("renders the first team option when includeAllTeams is false and when no selectedTeamId is given", () => {
    render(
      <TeamsDropdown
        currentUserTeams={USER_TEAMS}
        includeAllTeams={false}
        onChange={noop}
      />
    );

    const selectedTeam = screen.getByText("Team 1");
    expect(selectedTeam).toBeInTheDocument();
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

      const selectedTeam = screen.getByText("All fleets");
      expect(selectedTeam).toBeInTheDocument();
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

      const selectedTeam = screen.getByText("Team 1");
      expect(selectedTeam).toBeInTheDocument();
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

      expect(screen.getByText("Team 1")).toBeInTheDocument();
    });
  });

  describe("in-menu search", () => {
    it("renders a search input when the menu is opened", async () => {
      const user = userEvent.setup();
      render(
        <TeamsDropdown
          currentUserTeams={USER_TEAMS}
          selectedTeamId={1}
          onChange={noop}
        />
      );

      await user.click(screen.getByText("Team 1"));
      expect(screen.getByPlaceholderText("Search fleets")).toBeInTheDocument();
    });

    it("filters options by the search query", async () => {
      const user = userEvent.setup();
      render(
        <TeamsDropdown
          currentUserTeams={USER_TEAMS}
          selectedTeamId={1}
          onChange={noop}
        />
      );

      await user.click(screen.getByText("Team 1"));
      // Use fireEvent.change instead of user.type — user.type re-clicks the
      // input, which under jsdom moves focus off react-select's internal
      // input and closes the menu. fireEvent.change fires onChange directly.
      fireEvent.change(screen.getByPlaceholderText("Search fleets"), {
        target: { value: "Team 2" },
      });

      expect(screen.getByText("Team 2")).toBeInTheDocument();
      expect(screen.queryByText("All fleets")).not.toBeInTheDocument();
    });

    it("shows the empty-state message when nothing matches", async () => {
      const user = userEvent.setup();
      render(
        <TeamsDropdown
          currentUserTeams={USER_TEAMS}
          selectedTeamId={1}
          onChange={noop}
        />
      );

      await user.click(screen.getByText("Team 1"));
      fireEvent.change(screen.getByPlaceholderText("Search fleets"), {
        target: { value: "nothing-matches-this" },
      });

      expect(screen.getByText("No matching fleets")).toBeInTheDocument();
    });
  });

  describe("Add fleet button", () => {
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

      await user.click(screen.getByText("Team 1"));
      expect(
        screen.queryByRole("button", { name: /add fleet/i })
      ).not.toBeInTheDocument();
    });

    it("renders for global admins and navigates on click", async () => {
      const user = userEvent.setup();
      renderWithAppContext(
        <TeamsDropdown
          currentUserTeams={USER_TEAMS}
          selectedTeamId={1}
          onChange={noop}
        />,
        { contextValue: { isGlobalAdmin: true } }
      );

      await user.click(screen.getByText("Team 1"));
      const addButton = screen.getByRole("button", { name: /add fleet/i });
      expect(addButton).toBeInTheDocument();

      await user.click(addButton);
      expect(mockPush).toHaveBeenCalledWith("/settings/fleets");
    });
  });
});
