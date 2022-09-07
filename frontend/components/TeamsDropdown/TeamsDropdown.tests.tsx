import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";

import TeamsDropdown from "./TeamsDropdown";

describe("TeamsDropdown - component", () => {
  const USER_TEAMS = [
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

  it("renders 'All teams' when no selectedTeamId is given", () => {
    render(<TeamsDropdown currentUserTeams={USER_TEAMS} onChange={noop} />);

    const selectedTeam = screen.getByText("All teams");
    expect(selectedTeam).toBeInTheDocument();
  });

  it("renders the first team option when includeAll is false and when no selectedTeamId is given", () => {
    render(
      <TeamsDropdown
        currentUserTeams={USER_TEAMS}
        includeAll={false}
        onChange={noop}
      />
    );

    const selectedTeam = screen.getByText("Team 1");
    expect(selectedTeam).toBeInTheDocument();
  });
});
