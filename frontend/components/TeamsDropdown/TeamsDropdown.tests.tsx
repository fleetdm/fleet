import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
// TODOL Replace renderWithAppContext with createCustomRenderer
import { renderWithAppContext } from "test/test-utils";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";

import TeamsDropdown from "./TeamsDropdown";

describe("TeamsDropdown - component", () => {
  const USER_TEAMS = [
    { id: -1, name: "All teams" },
    { id: 1, name: "Team 1" },
    { id: 2, name: "Team 2" },
  ];

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

  it("renders the first team option when includeAll is false and when no selectedTeamId is given", () => {
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

    it("renders 'All teams' when no selectedTeamId is given", () => {
      renderWithAppContext(
        <TeamsDropdown currentUserTeams={USER_TEAMS} onChange={noop} />,
        { contextValue }
      );

      const selectedTeam = screen.getByText("All teams");
      expect(selectedTeam).toBeInTheDocument();
    });

    it("renders the first team option when includeAll is false and when no selectedTeamId is given", () => {
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
});
