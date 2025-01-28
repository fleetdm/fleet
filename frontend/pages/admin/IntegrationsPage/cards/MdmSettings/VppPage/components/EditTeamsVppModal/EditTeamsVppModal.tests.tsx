import {
  APP_CONTEXT_ALL_TEAMS_ID,
  APP_CONTEXT_NO_TEAM_ID,
} from "interfaces/team";
import {
  getOptions,
  selectedValueFromToken,
  teamIdsFromSelectedValue,
  updateSelectedValue,
} from "./EditTeamsVppModal";

describe("EditTeamsVppModal", () => {
  const allTeamsToken = {
    id: 1,
    org_name: "Org 1",
    location: "https://example.com/mdm/apple/mdm",
    renew_date: "2024-11-29T00:00:00Z",
    teams: [], // all teams
  };

  const unassignedToken = {
    id: 2,
    org_name: "Org 2",
    location: "https://example.com/mdm/apple/mdm",
    renew_date: "2024-11-29T00:00:00Z",
    teams: null, // unassigned
  };

  const noTeamToken = {
    id: 3,
    org_name: "Org 3",
    location: "https://example.com/mdm/apple/mdm",
    renew_date: "2024-11-29T00:00:00Z",
    teams: [{ team_id: 0, name: "No team" }],
  };

  const piratesAndNinjasToken = {
    id: 4,
    org_name: "Org 4",
    location: "https://example.com/mdm/apple/mdm",
    renew_date: "2024-11-29T00:00:00Z",
    teams: [
      { team_id: 2, name: "Pirates" },
      { team_id: 1, name: "Ninjas" },
    ],
  };

  const pandasToken = {
    id: 5,
    org_name: "Org 5",
    location: "https://example.com/mdm/apple/mdm",
    renew_date: "2024-11-29T00:00:00Z",
    teams: [{ team_id: 3, name: "Pandas" }],
  };

  const availableTeams = [
    { id: APP_CONTEXT_ALL_TEAMS_ID, name: "All teams" },
    { id: APP_CONTEXT_NO_TEAM_ID, name: "No team" },
    { id: 1, name: "Ninjas" },
    { id: 2, name: "Pirates" },
    { id: 3, name: "Pandas" },
    { id: 4, name: "Penguins" },
  ];

  const allOptions = availableTeams.map((t) => ({
    label: t.name,
    value: t.id,
  }));

  describe("getOptions", () => {
    it("returns no options when another token is all teams", () => {
      const tokens = [allTeamsToken, piratesAndNinjasToken];
      const currentToken = piratesAndNinjasToken;
      const options = getOptions(availableTeams, tokens, currentToken);
      expect(options).toEqual([]);
    });

    it("includes all options when all tokens are unassigned", () => {
      const tokens = [
        unassignedToken,
        { ...unassignedToken, id: 1337 },
        { ...unassignedToken, id: 1338 },
      ];
      const currentToken = unassignedToken;
      const options = getOptions(availableTeams, tokens, currentToken);
      expect(options).toEqual(allOptions);
    });

    it("includes all options when current token is all teams and other tokens are unassigned", () => {
      const tokens = [
        allTeamsToken,
        unassignedToken,
        { ...unassignedToken, id: 1337 },
        { ...unassignedToken, id: 1338 },
      ];
      const currentToken = allTeamsToken;
      const options = getOptions(availableTeams, tokens, currentToken);
      expect(options).toEqual(allOptions);
    });

    it("excludes all teams option when any token is assigned", () => {
      const tokens = [unassignedToken, piratesAndNinjasToken];
      const currentToken = unassignedToken;
      const options = getOptions(availableTeams, tokens, currentToken);
      expect(options).toEqual(
        options.filter((o) => o.value !== APP_CONTEXT_ALL_TEAMS_ID)
      );
    });

    it("excludes teams assigned to other tokens", () => {
      const tokens = [
        piratesAndNinjasToken,
        noTeamToken,
        pandasToken,
        unassignedToken,
      ];

      // test with unassignedToken
      expect(getOptions(availableTeams, tokens, unassignedToken)).toEqual([
        { label: "Penguins", value: 4 }, // only penguins is available
      ]);

      // test with piratesAndNinjasToken
      let unavailableTeamIds = [
        APP_CONTEXT_ALL_TEAMS_ID, // all teams is excluded unless current token is all teams or all tokens are unassigned
        APP_CONTEXT_NO_TEAM_ID, // already assigned to noTeamToken
        3, // already assigned to pandasToken
      ];
      expect(getOptions(availableTeams, tokens, piratesAndNinjasToken)).toEqual(
        allOptions.filter((o) => !unavailableTeamIds.includes(o.value))
      );

      // test with pandasToken
      unavailableTeamIds = [
        APP_CONTEXT_ALL_TEAMS_ID, // all teams is excluded unless current token is all teams or all tokens are unassigned
        APP_CONTEXT_NO_TEAM_ID, // already assigned to noTeamToken
        1, // already assigned to piratesAndNinjasToken
        2, // already assigned to piratesAndNinjasToken
      ];
      expect(getOptions(availableTeams, tokens, pandasToken)).toEqual(
        allOptions.filter((o) => !unavailableTeamIds.includes(o.value))
      );

      // test with noTeamToken
      unavailableTeamIds = [
        APP_CONTEXT_ALL_TEAMS_ID, // all teams is excluded unless current token is all teams or all tokens are unassigned
        1, // already assigned to piratesAndNinjasToken
        2, // already assigned to piratesAndNinjasToken
        3, // already assigned to pandasToken
      ];
      expect(getOptions(availableTeams, tokens, noTeamToken)).toEqual(
        allOptions.filter((o) => !unavailableTeamIds.includes(o.value))
      );

      // test with allTeamsToken
      unavailableTeamIds = [
        APP_CONTEXT_NO_TEAM_ID, // already assigned to noTeamToken
        1, // already assigned to piratesAndNinjasToken
        2, // already assigned to piratesAndNinjasToken
        3, // already assigned to pandasToken
      ];
      expect(
        getOptions(availableTeams, [...tokens, allTeamsToken], allTeamsToken)
      ).toEqual(
        allOptions.filter((o) => !unavailableTeamIds.includes(o.value))
      );
    });
  });
  describe("updateSelectedValue", () => {
    it("returns next value when only one team is selected", () => {
      const prev = "1";
      const next = "2";
      expect(updateSelectedValue(prev, next)).toEqual(next);
    });

    it("removes all teams when all teams was previously selected", () => {
      const prev = "-1,2";
      const next = "2";
      expect(updateSelectedValue(prev, next)).toEqual(next);
    });

    it("selects all teams when all teams is newly selected", () => {
      const prev = "2";
      const next = "2,-1";
      expect(updateSelectedValue(prev, next)).toEqual("-1");
    });

    it("returns next value when all teams is not selected", () => {
      const prev = "2";
      const next = "2,3";
      expect(updateSelectedValue(prev, next)).toEqual(next);
    });
  });

  describe("selectedValueFromToken", () => {
    it("returns empty string when teams is null", () => {
      expect(selectedValueFromToken(unassignedToken)).toEqual("");
    });

    it("returns all teams when teams is empty", () => {
      expect(selectedValueFromToken(allTeamsToken)).toEqual("-1");
    });

    it("returns team ids when teams is not empty", () => {
      expect(selectedValueFromToken(piratesAndNinjasToken)).toEqual("2,1");
    });
  });

  describe("teamIdsFromSelectedValue", () => {
    it("returns empty array when value is all teams id", () => {
      expect(
        teamIdsFromSelectedValue(APP_CONTEXT_ALL_TEAMS_ID.toString())
      ).toEqual([]);
    });

    it("returns mull when value is empty string", () => {
      expect(teamIdsFromSelectedValue("")).toEqual(null);
    });

    it("returns team ids when value is not all teams id", () => {
      expect(teamIdsFromSelectedValue("2,1")).toEqual([2, 1]);
    });
  });
});
