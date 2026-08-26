import React from "react";
import { act, screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import createMockUser from "__mocks__/userMock";
import {
  createMockSoftwareTitle,
  createMockSoftwarePackage,
  createMockAppStoreApp,
} from "__mocks__/softwareMock";

import { IPolicy } from "interfaces/policy";
import { ISoftwareTitle } from "interfaces/software";

import PolicyAutomationsFields, {
  IPolicyAutomationsFieldsHandle,
} from "./PolicyAutomationsFields";
import useSoftwareTitles from "./hooks/useSoftwareTitles";
import useScripts from "./hooks/useScripts";
import useProfiles from "./hooks/useProfiles";

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
  team_id: 1,
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
  software_package: createMockSoftwarePackage({
    installer_id: 100,
    name: "single-app.pkg",
    version: "1.0.0",
    uploaded_at: "2026-06-01T00:00:00Z",
  }),
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
  software_package: createMockSoftwarePackage({
    installer_id: 200,
    name: "multi-app-1.0.0.pkg",
    version: "1.0.0",
    uploaded_at: "2026-06-01T00:00:00Z",
  }),
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
  software_package: null,
  app_store_app: createMockAppStoreApp({ version: "5.0.0" }),
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
      software_title_id: 42,
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
      software_title_id: 42,
      patch_when_closed: false,
      continuous_automations_enabled: false,
    });
  });

  it("maps End user initiated to no continuous automation", () => {
    const handleRef: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null> = {
      current: null,
    };
    renderWithHandle(
      {
        type: "patch",
        patch_software: { name: "Firefox", software_title_id: 42 },
        install_software: { name: "Firefox", software_title_id: 42 },
        patch_when_closed: false,
        continuous_automations_enabled: true,
      },
      handleRef,
      { patchOption: "manual" }
    );

    expect(
      handleRef.current?.getAutomationsPayload().policyUpdate
    ).toMatchObject({
      software_title_id: null,
      continuous_automations_enabled: false,
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
      ".policy-automations-fields__section:last-child .fleet-checkbox__icon"
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
});

describe("PolicyAutomationsFields — Resend configuration profile row", () => {
  beforeEach(() => {
    mockedUseScripts.mockReturnValue(emptyScriptsResponse);
    setSoftwareTitles([]);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it("is enabled for platforms that support configuration profiles", () => {
    renderWithHandle(undefined, undefined, { selectedPlatforms: ["windows"] });

    expect(
      screen.getByRole("checkbox", { name: "resend_configuration_profile" })
    ).toHaveAttribute("aria-disabled", "false");
  });

  it("is disabled for platforms without configuration profiles", () => {
    renderWithHandle(undefined, undefined, { selectedPlatforms: ["linux"] });

    expect(
      screen.getByRole("checkbox", { name: "resend_configuration_profile" })
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
      screen.getByRole("checkbox", { name: "resend_configuration_profile" })
    ).toHaveAttribute("aria-checked", "true");
    expect(screen.getByText("Safari home page")).toBeInTheDocument();
  });

  it("blocks the save when the row is checked but no profile is selected", async () => {
    const handleRef: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null> = {
      current: null,
    };
    const { user } = renderWithHandle(undefined, handleRef, {
      selectedPlatforms: ["darwin"],
    });

    await user.click(
      screen.getByRole("checkbox", { name: "resend_configuration_profile" })
    );

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
