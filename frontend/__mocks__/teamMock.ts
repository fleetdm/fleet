import { ITeam, ITeamSummary } from "interfaces/team";

const DEFAULT_MOCK_TEAM_SUMMARY: ITeamSummary = {
  id: 1,
  name: "Team 1",
};

const DEFAUT_TEAM_MOCK: ITeam = {
  ...DEFAULT_MOCK_TEAM_SUMMARY,
};

export const createMockTeamSummary = (
  overrides?: Partial<ITeamSummary>
): ITeamSummary => {
  return { ...DEFAULT_MOCK_TEAM_SUMMARY, ...overrides };
};

const createMockTeam = (overrides?: Partial<ITeam>): ITeam => {
  return { ...DEFAUT_TEAM_MOCK, ...overrides };
};
export default createMockTeam;
