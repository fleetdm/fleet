import React from "react";
import { screen } from "@testing-library/react";

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

// Hook mocks — keep the tests focused on component behavior (dropdown
// visibility, auto-select, payload shape) instead of react-query wiring.
jest.mock("./hooks/useSoftwareTitles");
jest.mock("./hooks/useScripts");
jest.mock("hooks/useGitOpsMode", () => ({
  __esModule: true,
  default: () => ({ gitOpsModeEnabled: false }),
}));

const mockedUseSoftwareTitles = useSoftwareTitles as jest.MockedFunction<
  typeof useSoftwareTitles
>;
const mockedUseScripts = useScripts as jest.MockedFunction<typeof useScripts>;

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
  handleRef?: React.MutableRefObject<IPolicyAutomationsFieldsHandle | null>
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
});

describe("PolicyAutomationsFields — payload", () => {
  beforeEach(() => {
    mockedUseScripts.mockReturnValue(emptyScriptsResponse);
    setSoftwareTitles([singlePackageTitle, multiPackageTitle, vppTitle]);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it("carries software_installer_id (auto-selected first-added) for a multi-package title", async () => {
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
    expect(payload?.policyUpdate?.software_installer_id).toBe(200);
    expect(payload?.policyUpdate?.software_title_id).toBe(20);
  });

  it("does not error on save for a VPP title (must-fix: previously required non-null software_installer_id even without packages[])", () => {
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
    // missing installer_id when the selected title has no packages[]. The
    // payload can still be dirty on legacy-load (form pre-fill logic); the
    // point of this test is that isValid stays true so the parent can save.
    expect(payload?.isValid).toBe(true);
    // Backend picks the VPP install target from software_title_id; we send
    // installer_id as null on the wire.
    expect(payload?.policyUpdate?.software_installer_id ?? null).toBeNull();
  });
});
