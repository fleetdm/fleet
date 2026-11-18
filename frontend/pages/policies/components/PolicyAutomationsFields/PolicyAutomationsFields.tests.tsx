import { act, screen } from "@testing-library/react";
import React from "react";

import { createMockScript } from "__mocks__/scriptMock";
import {
  createMockSoftwareTitle,
  createMockSoftwarePackage,
  createMockAppStoreAppVersion,
} from "__mocks__/softwareMock";
import createMockUser from "__mocks__/userMock";
import { IPolicy } from "interfaces/policy";
import { ISoftwareTitle } from "interfaces/software";
import { createCustomRenderer } from "test/test-utils";

import useProfiles from "./hooks/useProfiles";
import useScripts from "./hooks/useScripts";
import useSoftwareTitles from "./hooks/useSoftwareTitles";
import PolicyAutomationsFields, {
  IPolicyAutomationsFieldsHandle,
} from "./PolicyAutomationsFields";

jest.mock("./hooks/useSoftwareTitles");
jest.mock("./hooks/useScripts");
jest.mock("./hooks/useProfiles");
jest.mock("hooks/useGitOpsMode", () => ({
  __esModule: true,
  default: () => ({ gitOpsModeEnabled: false }),
}));

const mockedUseSoftwareTitles = useSoftwareTitles as jest.MockedFunction<
  typeof useSoftwareTitles
>;
const mockedUseScripts = useScripts as jest.MockedFunction<typeof useScripts>;
const mockedUseProfiles = useProfiles as jest.MockedFunction<
  typeof useProfiles
>;

const setSoftwareTitles = (titles: ISoftwareTitle[]) => {
  mockedUseSoftwareTitles.mockReturnValue({
    data: {
      count: titles.length,
      counts_updated_at: null,
      meta: { has_next_results: false, has_previous_results: false },
      software_titles: titles,
    },
  } as ReturnType<typeof useSoftwareTitles>);
};

const emptyScriptsResponse = ({
  data: {
    count: 0,
    scripts: [],
    meta: { has_next_results: false, has_previous_results: false },
  },
} as unknown) as ReturnType<typeof useScripts>;

const emptyProfilesResponse = ({
  data: {
    meta: { has_next_results: false, has_previous_results: false },
    profiles: [],
  },
} as unknown) as ReturnType<typeof useProfiles>;

// Every render mounts the profiles hook; each describe clears mocks after
// itself, so re-seed the default (empty) response before each test.
beforeEach(() => {
  mockedUseProfiles.mockReturnValue(emptyProfilesResponse);
});

const createMockPolicy = (overrides?: Partial<IPolicy>): IPolicy => ({
  id: 1,
  name: "Test policy",
  query: "SELECT 1;",
  description: "",
  author_id: 1,
  author_name: "Admin",
  author_email: "admin@example.com",
  resolution: "",
  platform: "darwin",
  fleet_id: 1,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
  critical: false,
  calendar_events_enabled: false,
  conditional_access_enabled: false,
  type: "dynamic",
  ...overrides,
});

// Titles used across tests
const singlePackageTitle: ISoftwareTitle = createMockSoftwareTitle({
  id: 10,
  name: "Single App",
  source: "apps",
  packages: [
    createMockSoftwarePackage({
      installer_id: 100,
      name: "single-app.pkg",
      version: "1.0.0",
      uploaded_at: "2026-06-01T00:00:00Z",
    }),
  ],
});

const multiPackageTitle: ISoftwareTitle = createMockSoftwareTitle({
  id: 20,
  name: "Multi App",
  source: "apps",
  packages: [
    createMockSoftwarePackage({
      installer_id: 201,
      name: "multi-app-2.0.0.pkg",
      version: "2.0.0",
      uploaded_at: "2026-06-15T00:00:00Z",
    }),
    // Out of order to prove `findFirstAddedPackage` picks by smallest id.
    createMockSoftwarePackage({
      installer_id: 200,
      name: "multi-app-1.0.0.pkg",
      version: "1.0.0",
      uploaded_at: "2026-06-01T00:00:00Z",
    }),
    createMockSoftwarePackage({
      installer_id: 202,
      name: "multi-app-3.0.0.pkg",
      version: "3.0.0",
      uploaded_at: "2026-06-20T00:00:00Z",
    }),
  ],
});

const vppTitle: ISoftwareTitle = createMockSoftwareTitle({
  id: 30,
  name: "VPP App",
  source: "apps",
  app_store_apps: [createMockAppStoreAppVersion({ version: "5.0.0" })],
  packages: null,
});

const render = createCustomRenderer({
  context: {
    app: {
      currentUser: createMockUser({ global_role: "admin" }),
      isGlobalAdmin: true,
      isPremiumTier: true,
    },
  },
});

const openAutomationOptions = async (user: {
  click: (element: Element) => Promise<void>;
}) => {
  await user.click(screen.getByRole("combobox", { name: "Automation" }));
  return Array.from(document.querySelectorAll(".react-select__option"));
};

const getAutomationOption = async (
  user: { click: (element: Element) => Promise<void> },
  label: string
) => {
  const option = (await openAutomationOptions(user)).find((o) =>
    o.textContent?.startsWith(label)
  );
  if (!option) throw new Error(`No "${label}" automation option`);
  return option;
};

const selectAutomation = async (
  user: { click: (element: Element) => Promise<void> },
  label: string
) => {
  await user.click(await getAutomationOption(user, label));
};

/** Renders the field, forwarding the passed-in ref directly to the
 * component's `useImperativeHandle` so tests can call
 * `getAutomationsPayload()` after auto-select effects settle. Passing the
 * ref directly (vs copying it in a useEffect) avoids stale-closure reads:
 * `useImperativeHandle` reassigns `ref.current` on every render, so the
 * external `handleRef` always sees the latest closure. */
const renderWithHandle = (
  policyOverrides?: Partial<IPolicy>,
  handleRef?: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null>,
  componentProps?: Partial<
    React.ComponentPropsWithoutRef<typeof PolicyAutomationsFields>
  >
) => {
  return render(
    <PolicyAutomationsFields
      ref={handleRef}
      policy={createMockPolicy(policyOverrides)}
      isGlobalPolicy={false}
      teamIdForApi={1}
      automationsConfig={undefined}
      globalConfig={undefined}
      fleetName="Test Fleet"
      selectedPlatforms={["darwin"]}
      {...componentProps}
    />
  );
};

describe("PolicyAutomationsFields — Install software row", () => {
  beforeEach(() => {
    mockedUseScripts.mockReturnValue(emptyScriptsResponse);
    setSoftwareTitles([singlePackageTitle, multiPackageTitle, vppTitle]);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it("does not render the Select software dropdown when Install software is off", () => {
    renderWithHandle();
    expect(
      screen.queryByRole("combobox", { name: /Select package/i })
    ).not.toBeInTheDocument();
    // The outer dropdown's accessible name comes from react-select's default;
    // easier to check that the placeholder isn't in the DOM.
    expect(screen.queryByText("Select software")).not.toBeInTheDocument();
  });

  it("surfaces the Select package dropdown for a multi-package title and auto-selects the first-added (smallest installer_id)", () => {
    renderWithHandle({
      automation_type: "software",
      install_software: {
        name: "Multi App",
        software_title_id: 20,
      },
    });

    // Multi-package title has 3 packages — second dropdown must render, and
    // its selected option should be `multi-app-1.0.0.pkg` (installer_id 200 —
    // smallest even though it's not first in the packages[] array).
    const selectPackage = screen.getByRole("combobox", {
      name: /Select package/i,
    });
    expect(selectPackage).toBeInTheDocument();
    expect(screen.getByText("multi-app-1.0.0.pkg")).toBeInTheDocument();
  });

  it("does not surface the Select package dropdown for a single-package title", () => {
    renderWithHandle({
      automation_type: "software",
      install_software: {
        name: "Single App",
        software_title_id: 10,
      },
    });

    expect(
      screen.queryByRole("combobox", { name: /Select package/i })
    ).not.toBeInTheDocument();
  });

  it("does not surface the Select package dropdown for a VPP / App Store title (no packages[])", () => {
    renderWithHandle({
      automation_type: "software",
      install_software: {
        name: "VPP App",
        software_title_id: 30,
      },
    });

    expect(
      screen.queryByRole("combobox", { name: /Select package/i })
    ).not.toBeInTheDocument();
  });

  it("surfaces the Select package dropdown at the exact 2-package threshold and preselects the first-added package", async () => {
    // Pins the `packageOptions.length > 1` gate: two-package titles must
    // show the picker, one-package titles must not. Guards against a
    // future refactor accidentally moving the boundary to `>= 2` (same
    // effective behavior) but also against `> 2` (which would silently
    // hide the picker on the smallest multi-package title). Also confirms
    // the auto-select effect preselects the first-added (smallest
    // installer_id) — order-independent regardless of packages[] order.
    setSoftwareTitles([
      createMockSoftwareTitle({
        id: 40,
        name: "Duo App",
        source: "apps",
        packages: [
          // Out of order to prove first-added is picked by installer_id,
          // not by array position.
          createMockSoftwarePackage({
            installer_id: 401,
            name: "duo-app-2.0.0.pkg",
            version: "2.0.0",
            uploaded_at: "2026-06-15T00:00:00Z",
          }),
          createMockSoftwarePackage({
            installer_id: 400,
            name: "duo-app-1.0.0.pkg",
            version: "1.0.0",
            uploaded_at: "2026-06-01T00:00:00Z",
          }),
        ],
      }),
    ]);
    renderWithHandle({
      automation_type: "software",
      install_software: {
        name: "Duo App",
        software_title_id: 40,
      },
    });

    expect(
      screen.getByRole("combobox", { name: /Select package/i })
    ).toBeInTheDocument();
    // Auto-select is set by a useEffect (async post-commit); wait for the
    // preselected label to appear rather than reading state synchronously.
    expect(await screen.findByText("duo-app-1.0.0.pkg")).toBeInTheDocument();
  });
});

describe("PolicyAutomationsFields — payload", () => {
  beforeEach(() => {
    mockedUseScripts.mockReturnValue(emptyScriptsResponse);
    setSoftwareTitles([singlePackageTitle, multiPackageTitle, vppTitle]);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it("carries software_package_id (auto-selected first-added) for a multi-package title", async () => {
    const handleRef: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null> = {
      current: null,
    };
    renderWithHandle(
      {
        automation_type: "software",
        install_software: {
          name: "Multi App",
          software_title_id: 20,
        },
      },
      handleRef
    );

    // Wait for the auto-select useEffect to hydrate the second dropdown
    // (visible value = first-added filename) before reading the payload —
    // otherwise we're reading state from the initial commit, before the
    // effect has run.
    await screen.findByText("multi-app-1.0.0.pkg");

    const payload = handleRef.current?.getAutomationsPayload();
    expect(payload?.isValid).toBe(true);
    // First-added by smallest installer_id = 200
    expect(payload?.policyUpdate?.software_package_id).toBe(200);
    expect(payload?.policyUpdate?.software_title_id).toBe(20);
  });

  it("does not error on save for a VPP title (must-fix: previously required non-null software_package_id even without packages[])", () => {
    const handleRef: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null> = {
      current: null,
    };
    renderWithHandle(
      {
        automation_type: "software",
        install_software: {
          name: "VPP App",
          software_title_id: 30,
        },
      },
      handleRef
    );

    const payload = handleRef.current?.getAutomationsPayload();
    // Regression guard for the VPP path: validate() must NOT flag the
    // missing package_id when the selected title has no packages[]. The
    // payload can still be dirty on legacy-load (form pre-fill logic); the
    // point of this test is that isValid stays true so the parent can save.
    expect(payload?.isValid).toBe(true);
    // Backend picks the VPP install target from software_title_id; we send
    // package_id as null on the wire.
    expect(payload?.policyUpdate?.software_package_id ?? null).toBeNull();
  });

  it("maps Patch when app is closed to both policy flags", () => {
    const handleRef: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null> = {
      current: null,
    };
    renderWithHandle(
      {
        type: "patch",
        patch_software: { name: "Firefox", software_title_id: 42 },
        patch_when_closed: false,
        continuous_automations_enabled: false,
      },
      handleRef,
      { patchOption: "closed" }
    );

    expect(
      handleRef.current?.getAutomationsPayload().policyUpdate
    ).toMatchObject({
      automation_type: "software",
      patch_when_closed: true,
      continuous_automations_enabled: true,
    });
  });

  it("maps Force patch to patch_when_closed false and continuous automation off", () => {
    const handleRef: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null> = {
      current: null,
    };
    renderWithHandle(
      {
        type: "patch",
        patch_software: { name: "Firefox", software_title_id: 42 },
        patch_when_closed: true,
        continuous_automations_enabled: true,
      },
      handleRef,
      { patchOption: "force" }
    );

    // Switching a stored Patch when app is closed policy to Force patch clears
    // continuous automation instead of carrying the stored value over.
    expect(
      screen.getByRole("checkbox", { name: "continuous-automations-enabled" })
    ).toHaveAttribute("aria-checked", "false");

    expect(
      handleRef.current?.getAutomationsPayload().policyUpdate
    ).toMatchObject({
      automation_type: "software",
      patch_when_closed: false,
      continuous_automations_enabled: false,
    });
  });

  it("hides the continuous checkbox on End user initiated and preserves the stored value on the wire", () => {
    const handleRef: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null> = {
      current: null,
    };
    renderWithHandle(
      {
        type: "patch",
        patch_software: { name: "Firefox", software_title_id: 42 },
        install_software: { name: "Firefox", software_title_id: 42 },
        automation_type: "software",
        patch_when_closed: false,
        continuous_automations_enabled: true,
      },
      handleRef,
      { patchOption: "manual" }
    );

    // Manual hides the checkbox but preserves stored continuous — the wire
    // must not silently downgrade a value the user never touched.
    expect(
      handleRef.current?.getAutomationsPayload().policyUpdate
    ).toMatchObject({
      automation_type: null,
      continuous_automations_enabled: true,
    });

    expect(
      screen.queryByRole("checkbox", {
        name: "continuous-automations-enabled",
      })
    ).not.toBeInTheDocument();
  });

  it("checks and disables continuous automation for Patch when app is closed", async () => {
    const { user, container } = renderWithHandle(
      {
        type: "patch",
        patch_when_closed: true,
        continuous_automations_enabled: true,
      },
      undefined,
      { patchOption: "closed" }
    );

    const continuous = screen.getByRole("checkbox", {
      name: "continuous-automations-enabled",
    });
    expect(continuous).toHaveAttribute("aria-checked", "true");
    expect(continuous).toHaveAttribute("aria-disabled", "true");

    const icon = container.querySelector(
      ".policy-automations-fields__continuous .fleet-checkbox__icon"
    );
    expect(icon).not.toBeNull();
    await user.hover(icon as Element);
    expect(
      await screen.findByText(
        "Continuous automation can't be disabled when Patch when app is closed is selected."
      )
    ).toBeInTheDocument();
  });

  it("keeps continuous automation editable for Force patch", async () => {
    const { user } = renderWithHandle(
      {
        type: "patch",
        patch_when_closed: false,
        continuous_automations_enabled: false,
      },
      undefined,
      { patchOption: "force" }
    );

    const continuous = screen.getByRole("checkbox", {
      name: "continuous-automations-enabled",
    });
    expect(continuous).toHaveAttribute("aria-disabled", "false");
    expect(continuous).toHaveAttribute("aria-checked", "false");

    await user.click(continuous);
    expect(continuous).toHaveAttribute("aria-checked", "true");
  });

  it("does not clear continuous automation already stored on a Force patch policy", () => {
    renderWithHandle(
      {
        type: "patch",
        patch_when_closed: false,
        continuous_automations_enabled: true,
      },
      undefined,
      { patchOption: "force" }
    );

    expect(
      screen.getByRole("checkbox", { name: "continuous-automations-enabled" })
    ).toHaveAttribute("aria-checked", "true");
  });

  it("maps Force patch + Notify before patching to notify_before_patching:true + continuous on", () => {
    const handleRef: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null> = {
      current: null,
    };
    renderWithHandle(
      {
        type: "patch",
        platform: "darwin",
        patch_software: { name: "Firefox", software_title_id: 42 },
        patch_when_closed: false,
        notify_before_patching: false,
        continuous_automations_enabled: false,
      },
      handleRef,
      { patchOption: "force", endUserExperience: "notify" }
    );

    expect(
      handleRef.current?.getAutomationsPayload().policyUpdate
    ).toMatchObject({
      automation_type: "software",
      notify_before_patching: true,
      continuous_automations_enabled: true,
    });
  });

  it("locks continuous automation with the Notify tooltip when Notify before patching is selected", async () => {
    const { user, container } = renderWithHandle(
      {
        type: "patch",
        platform: "darwin",
        patch_when_closed: false,
        notify_before_patching: false,
        continuous_automations_enabled: false,
      },
      undefined,
      { patchOption: "force", endUserExperience: "notify" }
    );

    const continuous = screen.getByRole("checkbox", {
      name: "continuous-automations-enabled",
    });
    expect(continuous).toHaveAttribute("aria-checked", "true");
    expect(continuous).toHaveAttribute("aria-disabled", "true");

    const icon = container.querySelector(
      ".policy-automations-fields__continuous .fleet-checkbox__icon"
    );
    expect(icon).not.toBeNull();
    await user.hover(icon as Element);
    expect(
      await screen.findByText(
        "Continuous automation can't be disabled when Notify before patching is selected."
      )
    ).toBeInTheDocument();
  });
});

describe("PolicyAutomationsFields — Resend configuration profile row", () => {
  beforeEach(() => {
    mockedUseScripts.mockReturnValue(emptyScriptsResponse);
    setSoftwareTitles([]);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it("is enabled for platforms that support configuration profiles", async () => {
    const { user } = renderWithHandle(undefined, undefined, {
      selectedPlatforms: ["windows"],
    });

    expect(
      await getAutomationOption(user, "Resend configuration profile")
    ).toHaveAttribute("aria-disabled", "false");
  });

  it("is disabled for platforms without configuration profiles", async () => {
    const { user } = renderWithHandle(undefined, undefined, {
      selectedPlatforms: ["linux"],
    });

    expect(
      await getAutomationOption(user, "Resend configuration profile")
    ).toHaveAttribute("aria-disabled", "true");
  });

  it("keeps the stored selection while the platform checkboxes are still hydrating", () => {
    mockedUseProfiles.mockReturnValue(({
      data: {
        meta: { has_next_results: false, has_previous_results: false },
        profiles: [
          {
            profile_uuid: "abc-123",
            name: "Safari home page",
            platform: "darwin",
          },
        ],
      },
    } as unknown) as ReturnType<typeof useProfiles>);

    // PolicyForm's platform checkboxes mount unchecked and are filled in by an
    // effect, so the first render sees an empty selection (#51272).
    const { rerender } = renderWithHandle(
      {
        automation_type: "configuration_profile",
        resend_configuration_profile: {
          profile_uuid: "abc-123",
          name: "Safari home page",
        },
      },
      undefined,
      { selectedPlatforms: [] }
    );

    rerender(
      <PolicyAutomationsFields
        policy={createMockPolicy({
          automation_type: "configuration_profile",
          resend_configuration_profile: {
            profile_uuid: "abc-123",
            name: "Safari home page",
          },
        })}
        isGlobalPolicy={false}
        teamIdForApi={1}
        automationsConfig={undefined}
        globalConfig={undefined}
        fleetName="Test Fleet"
        selectedPlatforms={["darwin"]}
      />
    );

    expect(
      screen.getByText("Resend configuration profile")
    ).toBeInTheDocument();
    expect(screen.getByText("Safari home page")).toBeInTheDocument();
  });

  it("blocks the save when the automation is selected but no profile is", async () => {
    const handleRef: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null> = {
      current: null,
    };
    const { user } = renderWithHandle(undefined, handleRef, {
      selectedPlatforms: ["darwin"],
    });

    await selectAutomation(user, "Resend configuration profile");

    act(() => {
      expect(handleRef.current?.getAutomationsPayload().isValid).toBe(false);
    });
    expect(
      await screen.findByText(
        "Please select a configuration profile to resend."
      )
    ).toBeInTheDocument();
  });
});

describe("PolicyAutomationsFields — type-to-search pickers", () => {
  beforeEach(() => {
    setSoftwareTitles([singlePackageTitle, multiPackageTitle, vppTitle]);
    mockedUseScripts.mockReturnValue(({
      data: {
        count: 2,
        scripts: [
          createMockScript({ id: 1, name: "Rotate keys" }),
          createMockScript({ id: 2, name: "Clear cache" }),
        ],
        meta: { has_next_results: false, has_previous_results: false },
      },
    } as unknown) as ReturnType<typeof useScripts>);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it("filters software titles as the user types", async () => {
    const { user } = renderWithHandle({
      automation_type: "software",
      install_software: { name: "Single App", software_title_id: 10 },
    });

    await user.type(
      screen.getByRole("combobox", { name: /Select software/i }),
      "Multi"
    );

    const options = Array.from(
      document.querySelectorAll(".react-select__option")
    ).map((o) => o.textContent);
    expect(options).toHaveLength(1);
    expect(options[0]).toContain("Multi App");
  });

  it("filters scripts as the user types", async () => {
    const { user } = renderWithHandle({
      automation_type: "script",
      run_script: { id: 1, name: "Rotate keys" },
    });

    await user.type(
      screen.getByRole("combobox", { name: /Select script/i }),
      "Clear"
    );

    const options = Array.from(
      document.querySelectorAll(".react-select__option")
    ).map((o) => o.textContent);
    expect(options).toEqual(["Clear cache"]);
  });
});

describe("PolicyAutomationsFields — Automation dropdown", () => {
  const webhookConfig = {
    webhook_settings: {
      failing_policies_webhook: {
        enable_failing_policies_webhook: true,
        destination_url: "https://example.com",
        policy_ids: [],
        host_batch_size: 0,
      },
    },
  } as never;

  beforeEach(() => {
    mockedUseScripts.mockReturnValue(({
      data: {
        count: 1,
        scripts: [createMockScript({ id: 1, name: "Rotate keys" })],
        meta: { has_next_results: false, has_previous_results: false },
      },
    } as unknown) as ReturnType<typeof useScripts>);
    setSoftwareTitles([singlePackageTitle]);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  const optionLabels = (options: Element[]) =>
    options.map((o) => o.textContent);

  it("shows None for a policy without an automation", () => {
    renderWithHandle();
    expect(screen.getByText("None")).toBeInTheDocument();
  });

  it("offers every automation for a fleet policy", async () => {
    const { user } = renderWithHandle(undefined, undefined, {
      automationsConfig: webhookConfig,
    });
    expect(optionLabels(await openAutomationOptions(user))).toEqual([
      "None",
      "Send webhook",
      "Install software",
      "Run script",
      "Resend configuration profile",
      "Calendar event",
      "Conditional access",
    ]);
  });

  it("only offers webhook or ticket for an All fleets policy", async () => {
    const { user } = renderWithHandle(undefined, undefined, {
      isGlobalPolicy: true,
      teamIdForApi: undefined,
      automationsConfig: webhookConfig,
    });
    expect(optionLabels(await openAutomationOptions(user))).toEqual([
      "None",
      "Send webhook",
    ]);
  });

  it("doesn't offer calendar for an Unassigned policy", async () => {
    const { user } = renderWithHandle(undefined, undefined, {
      teamIdForApi: 0,
    });
    expect(optionLabels(await openAutomationOptions(user))).not.toContain(
      "Calendar event"
    );
  });

  it("disables automations that aren't turned on for the fleet", async () => {
    const { user } = renderWithHandle();
    const options = await openAutomationOptions(user);
    const disabled = options
      .filter((o) => o.getAttribute("aria-disabled") === "true")
      .map((o) => o.textContent);
    expect(disabled).toEqual([
      "Send webhook or create ticket",
      "Calendar event",
      "Conditional access",
    ]);
  });

  it("shows the item dropdown and continuous automations for a script", async () => {
    const handleRef: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null> = {
      current: null,
    };
    const { user } = renderWithHandle(undefined, handleRef);

    await selectAutomation(user, "Run script");
    expect(screen.getByText("Script")).toBeInTheDocument();
    expect(
      screen.getByRole("checkbox", { name: "continuous-automations-enabled" })
    ).toBeInTheDocument();

    act(() => {
      expect(handleRef.current?.getAutomationsPayload().isValid).toBe(false);
    });
    expect(
      await screen.findByText("Please select a script to run.")
    ).toBeInTheDocument();
  });

  it("hides continuous automations for automations other than software and script", async () => {
    const { user } = renderWithHandle(undefined, undefined, {
      automationsConfig: webhookConfig,
    });

    await selectAutomation(user, "Send webhook");
    expect(
      screen.queryByRole("checkbox", { name: "continuous-automations-enabled" })
    ).not.toBeInTheDocument();
  });

  it("sends the selected automation with its item in one update", async () => {
    const handleRef: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null> = {
      current: null,
    };
    const { user } = renderWithHandle(
      {
        automation_type: "software",
        install_software: { name: "Single App", software_title_id: 10 },
      },
      handleRef
    );

    await selectAutomation(user, "Run script");
    await user.click(screen.getByRole("combobox", { name: /Select script/i }));
    await user.click(screen.getByText("Rotate keys"));

    expect(handleRef.current?.getAutomationsPayload()).toEqual({
      isValid: true,
      isDirty: true,
      policyUpdate: {
        automation_type: "script",
        script_id: 1,
        continuous_automations_enabled: false,
      },
    });
  });

  it("sends a null automation_type when None is selected", async () => {
    const handleRef: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null> = {
      current: null,
    };
    const { user } = renderWithHandle(
      { automation_type: "script", run_script: { id: 1, name: "Rotate keys" } },
      handleRef
    );

    await selectAutomation(user, "None");
    expect(
      handleRef.current?.getAutomationsPayload().policyUpdate
    ).toMatchObject({ automation_type: null });
  });

  it("sends nothing when the automation didn't change", () => {
    const handleRef: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null> = {
      current: null,
    };
    renderWithHandle(
      { automation_type: "script", run_script: { id: 1, name: "Rotate keys" } },
      handleRef
    );

    expect(handleRef.current?.getAutomationsPayload()).toEqual({
      isValid: true,
      isDirty: false,
    });
  });

  describe("for a maintainer", () => {
    const renderAsMaintainer = createCustomRenderer({
      context: {
        app: {
          currentUser: createMockUser({ global_role: "maintainer" }),
          isGlobalAdmin: false,
          isPremiumTier: true,
        },
      },
    });

    it("disables the webhook or ticket option", async () => {
      const { user } = renderAsMaintainer(
        <PolicyAutomationsFields
          policy={createMockPolicy()}
          isGlobalPolicy={false}
          teamIdForApi={1}
          automationsConfig={webhookConfig}
          globalConfig={undefined}
          fleetName="Test Fleet"
          selectedPlatforms={["darwin"]}
        />
      );

      expect(await getAutomationOption(user, "Send webhook")).toHaveAttribute(
        "aria-disabled",
        "true"
      );
    });

    it("locks the dropdown when the policy's automation is webhook or ticket", () => {
      renderAsMaintainer(
        <PolicyAutomationsFields
          policy={createMockPolicy({ automation_type: "webhook_or_ticket" })}
          isGlobalPolicy={false}
          teamIdForApi={1}
          automationsConfig={webhookConfig}
          globalConfig={undefined}
          fleetName="Test Fleet"
          selectedPlatforms={["darwin"]}
        />
      );

      expect(
        screen.getByRole("combobox", { name: "Automation" })
      ).toBeDisabled();
    });
  });
});
