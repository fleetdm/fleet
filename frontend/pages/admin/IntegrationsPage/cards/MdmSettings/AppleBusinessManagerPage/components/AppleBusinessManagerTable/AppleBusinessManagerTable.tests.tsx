import { IMdmAbToken } from "interfaces/mdm";

import { generateActions } from "./AppleBusinessManagerTableConfig";

const createToken = (overrides: Partial<IMdmAbToken> = {}): IMdmAbToken => ({
  id: 1,
  apple_id: "apple@example.com",
  org_name: "Acme Inc.",
  mdm_server_url: "https://example.com/mdm/apple/mdm",
  renew_date: "2027-05-29T00:00:00Z",
  terms_expired: false,
  token_invalid: false,
  default: false,
  macos_fleet: { name: "No team", fleet_id: 0 },
  ios_fleet: { name: "No team", fleet_id: 0 },
  ipados_fleet: { name: "No team", fleet_id: 0 },
  byod_fleet: { name: "No team", fleet_id: 0 },
  ...overrides,
});

describe("AppleBusinessManagerTable generateActions", () => {
  const findToggle = (actions: ReturnType<typeof generateActions>) =>
    actions.find((a) => a.value === "toggleDefault");

  it("offers 'Set as default token' for a non-default token", () => {
    const actions = generateActions(createToken(), 2, false);

    const toggle = findToggle(actions);
    expect(toggle?.label).toBe("Set as default token");
    expect(toggle?.disabled).toBe(false);
    expect(actions.map((a) => a.value)).toEqual([
      "editTeams",
      "toggleDefault",
      "renew",
      "delete",
    ]);
  });

  it("offers 'Unset default token' for the default token", () => {
    const actions = generateActions(createToken({ default: true }), 2, false);

    const toggle = findToggle(actions);
    expect(toggle?.label).toBe("Unset default token");
    expect(toggle?.disabled).toBe(false);
  });

  it("disables unsetting the default when it is the only token", () => {
    const actions = generateActions(createToken({ default: true }), 1, false);

    const toggle = findToggle(actions);
    expect(toggle?.label).toBe("Unset default token");
    expect(toggle?.disabled).toBe(true);
    expect(toggle?.tooltipContent).toBe(
      "The only AB token is always the default."
    );
  });

  it("disables set as default and edit fleets in GitOps mode", () => {
    const actions = generateActions(
      createToken(),
      2,
      true,
      "https://example.com/repo"
    );

    expect(findToggle(actions)?.disabled).toBe(true);
    expect(actions.find((a) => a.value === "editTeams")?.disabled).toBe(true);
    expect(actions.find((a) => a.value === "renew")?.disabled).toBe(false);
    expect(actions.find((a) => a.value === "delete")?.disabled).toBe(false);
  });
});
