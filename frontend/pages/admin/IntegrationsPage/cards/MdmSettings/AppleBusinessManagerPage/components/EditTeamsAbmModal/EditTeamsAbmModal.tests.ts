import {
  APP_CONTEXT_NO_TEAM_SUMMARY,
  APP_CONTEXT_ALL_TEAMS_SUMMARY,
} from "interfaces/team";
import { getOptions, getSelectedTeamIds } from "./EditTeamsAbmModal";

describe("EditTeamsAbmModal", () => {
  const availableTeams = [
    APP_CONTEXT_ALL_TEAMS_SUMMARY,
    APP_CONTEXT_NO_TEAM_SUMMARY,
    { name: "Team 1", id: 1 },
    { name: "Team 2", id: 2 },
  ];

  describe("getOptions", () => {
    it("excludes all teams", () => {
      const expectedOptions = availableTeams.reduce((acc, t) => {
        if (t.name !== "All teams") {
          acc.push({ value: t.name, label: t.name });
        }
        return acc;
      }, [] as { value: string; label: string }[]);
      expect(getOptions(availableTeams)).toEqual(expectedOptions);
    });
  });

  describe("getSelectedTeamIds", () => {
    it("returns selected team ids", () => {
      const selectedTeamNames = {
        ios_team: "Team 1",
        ipados_team: "Team 2",
        macos_team: "No team",
      };
      expect(getSelectedTeamIds(selectedTeamNames, availableTeams)).toEqual({
        ios_team_id: 1,
        ipados_team_id: 2,
        macos_team_id: 0,
      });
    });
  });
});
