import { rebuildQueryStringWithTeamId } from "hooks/useTeamIdParam";
import {
  APP_CONTEXT_ALL_TEAMS_ID,
  APP_CONTEXT_NO_TEAM_ID,
} from "interfaces/team";

import { STRIP_FLEET_SCOPED_FILTERS_ON_ALL_FLEETS } from "./helpers";

describe("STRIP_FLEET_SCOPED_FILTERS_ON_ALL_FLEETS", () => {
  const rebuild = (query: string, newTeamId: number) =>
    rebuildQueryStringWithTeamId(
      query,
      newTeamId,
      2,
      STRIP_FLEET_SCOPED_FILTERS_ON_ALL_FLEETS
    );

  it.each([
    "os_settings=pending",
    "apple_settings=failing",
    "macos_settings=latest",
    "os_settings_disk_encryption=verified",
    "macos_bootstrap_package=failed",
    "bootstrap_package=pending",
    "software_status=failed",
  ])("drops %s when switching to All fleets", (filter) => {
    expect(
      rebuild(
        `?fleet_id=2&${filter}&query=mac&order_key=name`,
        APP_CONTEXT_ALL_TEAMS_ID
      )
    ).toBe("?query=mac&order_key=name");
  });

  it("keeps the filter when switching to another fleet", () => {
    expect(rebuild("?fleet_id=2&os_settings=pending", 3)).toBe(
      "?fleet_id=3&os_settings=pending"
    );
  });

  it("keeps the filter when switching to No fleet", () => {
    expect(
      rebuild("?fleet_id=2&os_settings=pending", APP_CONTEXT_NO_TEAM_ID)
    ).toBe("?fleet_id=0&os_settings=pending");
  });
});
