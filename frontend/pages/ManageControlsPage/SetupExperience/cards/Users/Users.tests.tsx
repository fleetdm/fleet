import { IMdmAbmToken } from "interfaces/mdm";

import { isAdeConfiguredForTeam } from "./Users";

const createAbmToken = (
  overrides: Partial<{
    macosTeamId: number;
    iosTeamId: number;
    ipadosTeamId: number;
  }> = {}
): IMdmAbmToken => ({
  id: 1,
  apple_id: "test@example.com",
  org_name: "Test Org",
  mdm_server_url: "https://mdm.example.com",
  renew_date: "2027-01-01T00:00:00Z",
  terms_expired: false,
  macos_team: { team_id: overrides.macosTeamId ?? 1, name: "Workstations" },
  ios_team: { team_id: overrides.iosTeamId ?? 2, name: "iOS devices" },
  ipados_team: {
    team_id: overrides.ipadosTeamId ?? 2,
    name: "iPadOS devices",
  },
});

describe("isAdeConfiguredForTeam", () => {
  it("returns false when abmTokens is undefined", () => {
    expect(isAdeConfiguredForTeam(1, undefined)).toBe(false);
  });

  it("returns false when abmTokens is empty", () => {
    expect(isAdeConfiguredForTeam(1, [])).toBe(false);
  });

  it("returns true when team matches a macos_team", () => {
    const tokens = [createAbmToken({ macosTeamId: 5 })];
    expect(isAdeConfiguredForTeam(5, tokens)).toBe(true);
  });

  it("returns true when team matches an ios_team", () => {
    const tokens = [createAbmToken({ iosTeamId: 7 })];
    expect(isAdeConfiguredForTeam(7, tokens)).toBe(true);
  });

  it("returns true when team matches an ipados_team", () => {
    const tokens = [createAbmToken({ ipadosTeamId: 9 })];
    expect(isAdeConfiguredForTeam(9, tokens)).toBe(true);
  });

  it("returns false when team does not match any token", () => {
    const tokens = [createAbmToken()];
    expect(isAdeConfiguredForTeam(999, tokens)).toBe(false);
  });

  it("returns true when team matches in one of multiple tokens", () => {
    const tokens = [
      createAbmToken({ macosTeamId: 1 }),
      createAbmToken({ macosTeamId: 10 }),
    ];
    expect(isAdeConfiguredForTeam(10, tokens)).toBe(true);
  });

  it("handles 'No team' (id 0) correctly", () => {
    const tokens = [createAbmToken({ macosTeamId: 0 })];
    expect(isAdeConfiguredForTeam(0, tokens)).toBe(true);
  });

  it("returns false for 'No team' (id 0) when no token assigns it", () => {
    const tokens = [createAbmToken({ macosTeamId: 1 })];
    expect(isAdeConfiguredForTeam(0, tokens)).toBe(false);
  });
});
