import { getAutomationsForPolicy, getPoliciesFilterParams } from "./helpers";

describe("getPoliciesFilterParams", () => {
  it("sends no filter when none is selected", () => {
    expect(getPoliciesFilterParams(null)).toEqual({});
  });

  it("filters patch policies by type", () => {
    expect(getPoliciesFilterParams("patch")).toEqual({ type: "patch" });
  });

  it("filters the other options by automation type", () => {
    expect(getPoliciesFilterParams("webhook_or_ticket")).toEqual({
      automationType: "webhook_or_ticket",
    });
    expect(getPoliciesFilterParams("configuration_profile")).toEqual({
      automationType: "configuration_profile",
    });
  });
});

describe("getAutomationsForPolicy", () => {
  const basePolicy = { automation_type: null };
  const software = { ...basePolicy, automation_type: "software" as const };

  it("returns empty array when no automations are enabled", () => {
    expect(getAutomationsForPolicy(basePolicy)).toEqual([]);
  });

  it("returns software automation with display_name preferred over name", () => {
    const result = getAutomationsForPolicy({
      ...software,
      install_software: {
        name: "Chrome.app",
        display_name: "Google Chrome",
        software_title_id: 42,
      },
    });
    expect(result).toHaveLength(1);
    expect(result[0]).toMatchObject({
      type: "software",
      name: "Google Chrome",
      softwareTitleId: 42,
    });
  });

  it("carries the custom icon_url onto the software automation", () => {
    const result = getAutomationsForPolicy({
      ...software,
      install_software: {
        name: "Chrome.app",
        software_title_id: 42,
        icon_url: "/api/latest/fleet/software/titles/42/icon?fleet_id=1",
      },
    });
    expect(result[0]).toMatchObject({
      type: "software",
      iconUrl: "/api/latest/fleet/software/titles/42/icon?fleet_id=1",
    });
  });

  it("leaves iconUrl undefined when no custom icon exists", () => {
    const result = getAutomationsForPolicy({
      ...software,
      install_software: { name: "Chrome.app", software_title_id: 42 },
    });
    expect(result[0]).toMatchObject({ type: "software" });
    expect((result[0] as { iconUrl?: string | null }).iconUrl).toBeUndefined();
  });

  it("falls back to name when display_name is absent", () => {
    const result = getAutomationsForPolicy({
      ...software,
      install_software: { name: "Chrome.app", software_title_id: 42 },
    });
    expect(result[0].name).toBe("Chrome.app");
  });

  it("normalizes known awkward titles via getDisplayedSoftwareName", () => {
    const result = getAutomationsForPolicy({
      ...software,
      install_software: {
        name: "Microsoft.CompanyPortal",
        software_title_id: 42,
      },
    });
    expect(result[0].name).toBe("Company Portal");
  });

  it("preserves the raw install_software.name as iconName for fallback icon matching (regression: #47123)", () => {
    const result = getAutomationsForPolicy({
      ...software,
      install_software: {
        name: "Zoom",
        display_name: "Custom Renamed App",
        software_title_id: 42,
      },
    });
    expect(result[0]).toMatchObject({
      type: "software",
      name: "Custom Renamed App",
      iconName: "Zoom",
    });
  });

  it("returns script automation with file name", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      automation_type: "script",
      run_script: { id: 7, name: "fix-disk.sh" },
    });
    expect(result).toHaveLength(1);
    expect(result[0]).toMatchObject({
      type: "script",
      name: "fix-disk.sh",
    });
  });

  it("returns calendar automation", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      automation_type: "calendar",
    });
    expect(result[0]).toMatchObject({
      type: "calendar",
      name: "Maintenance window",
    });
  });

  it("returns conditional access automation", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      automation_type: "conditional_access",
    });
    expect(result[0]).toMatchObject({
      type: "conditional_access",
      name: "Conditional access",
    });
  });

  it("labels other automation as Webhook by default", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      automation_type: "webhook_or_ticket",
    });
    expect(result[0]).toMatchObject({ type: "other", name: "Webhook" });
  });

  it("labels other automation as Ticket when otherAutomationType is ticket", () => {
    const result = getAutomationsForPolicy(
      { ...basePolicy, automation_type: "webhook_or_ticket" },
      "ticket"
    );
    expect(result[0]).toMatchObject({ type: "other", name: "Ticket" });
  });

  it("does not include other automation when the policy has none", () => {
    const result = getAutomationsForPolicy(basePolicy);
    expect(result).toHaveLength(0);
  });

  it("returns only the policy's automation", () => {
    const result = getAutomationsForPolicy(
      {
        automation_type: "script",
        install_software: { name: "Chrome.app", software_title_id: 1 },
        run_script: { id: 2, name: "fix.sh" },
      },
      "webhook"
    );
    expect(result.map((a) => a.type)).toEqual(["script"]);
  });
});
