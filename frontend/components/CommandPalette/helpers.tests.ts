import createMockConfig from "__mocks__/configMock";

import { buildPaletteItems, GROUPS, ICommandPaletteContext } from "./helpers";

const BASE_CONTEXT: ICommandPaletteContext = {
  search: "",
  currentTeam: undefined,
  config: createMockConfig(),
  canAccessControls: true,
  canWrite: true,
  canRunLiveReport: true,
  canAccessSettings: true,
  canManagePolicyAutomations: true,
  canManageSoftwareAutomations: true,
  isTechnician: false,
  isPremiumTier: true,
  isMacMdmEnabledAndConfigured: true,
  isWindowsMdmEnabledAndConfigured: true,
  isAndroidMdmEnabledAndConfigured: false,
  isVppEnabled: false,
  hasTeamSelected: false,

  withTeamId: (path: string) => path,
  onToggleDarkMode: jest.fn(),
  onViewHost: jest.fn(),
  onViewSoftware: jest.fn(),
  onViewSoftwareLibrary: jest.fn(),
  onViewReport: jest.fn(),
  onViewPolicy: jest.fn(),
};

describe("CommandPalette helpers", () => {
  describe("GROUPS", () => {
    it("contains all expected groups in order", () => {
      expect(GROUPS).toEqual([
        "Pages",
        "Controls",
        "Software",
        "Settings",
        "MDM",
        "Automations",
        "Commands",
      ]);
    });
  });

  describe("buildPaletteItems", () => {
    it("returns items for a global admin with a team selected", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: true,
        currentTeam: { id: 1, name: "Engineering" },
      });
      expect(items.length).toBeGreaterThan(0);

      const ids = items.map((i) => i.id);
      expect(ids).toContain("dashboard");
      expect(ids).toContain("hosts");
      expect(ids).toContain("controls-page");
      expect(ids).toContain("software-page");
      expect(ids).toContain("reports");
      expect(ids).toContain("policies");
      expect(ids).toContain("settings-page");
    });

    it("hides controls on All fleets", () => {
      const items = buildPaletteItems(BASE_CONTEXT);
      const ids = items.map((i) => i.id);
      expect(ids).not.toContain("controls-page");
      expect(ids).not.toContain("controls-os-updates");
      expect(ids).not.toContain("controls-os-settings");
    });

    it("excludes controls for observers", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        canAccessControls: false,
        canWrite: false,
        canAccessSettings: false,
        canManagePolicyAutomations: false,
        canManageSoftwareAutomations: false,
      });

      const ids = items.map((i) => i.id);
      expect(ids).not.toContain("controls-page");
      expect(ids).not.toContain("controls-os-updates");
      expect(ids).not.toContain("settings-page");
      expect(ids).not.toContain("add-hosts");
    });

    it("excludes settings for non-global-admins", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        canAccessSettings: false,
        canManageSoftwareAutomations: false,
      });

      const ids = items.map((i) => i.id);
      expect(ids).not.toContain("settings-page");
      expect(ids).not.toContain("settings-organization");
      expect(ids).not.toContain("settings-integrations");
      expect(ids).not.toContain("manage-software-automations");
    });

    it("shows packs only when searching for 'packs'", () => {
      const itemsNoSearch = buildPaletteItems(BASE_CONTEXT);
      expect(itemsNoSearch.map((i) => i.id)).not.toContain("packs");

      const itemsWithSearch = buildPaletteItems({
        ...BASE_CONTEXT,
        search: "packs",
      });
      const ids = itemsWithSearch.map((i) => i.id);
      expect(ids).toContain("packs");
      expect(ids).toContain("new-pack");
    });

    it("does not show packs when searching for 'package'", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        search: "package",
      });
      expect(items.map((i) => i.id)).not.toContain("packs");
    });

    it("omits the teamName chip when destination matches current context", () => {
      // On Engineering, every action either stays on Engineering or goes
      // there — no chip should render.
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: true,
        currentTeam: { id: 1, name: "Engineering" },
      });

      const addHosts = items.find((i) => i.id === "add-hosts");
      expect(addHosts?.teamName).toBeUndefined();

      const addReport = items.find((i) => i.id === "add-report");
      expect(addReport?.teamName).toBeUndefined();
    });

    it("shows 'Unassigned' on add-hosts and manage-enroll-secrets when on All fleets", () => {
      const items = buildPaletteItems(BASE_CONTEXT);

      const addHosts = items.find((i) => i.id === "add-hosts");
      expect(addHosts?.teamName).toBe("Unassigned");

      const enrollSecrets = items.find((i) => i.id === "manage-enroll-secrets");
      expect(enrollSecrets?.teamName).toBe("Unassigned");
    });

    it("omits the 'All fleets' chip on default-context actions when already on All fleets", () => {
      // add-report stays on All fleets when invoked from All fleets — no
      // switch, no chip.
      const items = buildPaletteItems(BASE_CONTEXT);
      const addReport = items.find((i) => i.id === "add-report");
      expect(addReport?.teamName).toBeUndefined();
    });

    it("shows 'All fleets' on default-context actions when on Unassigned", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: false,
        currentTeam: { id: 0, name: "No team" },
      });

      const addReport = items.find((i) => i.id === "add-report");
      expect(addReport?.teamName).toBe("All fleets");
    });

    it("omits the 'Unassigned' chip on add-hosts when already on Unassigned", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: false,
        currentTeam: { id: 0, name: "No team" },
      });

      const addHosts = items.find((i) => i.id === "add-hosts");
      expect(addHosts?.teamName).toBeUndefined();
    });

    it("shows 'Turn on' MDM when not configured", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        isMacMdmEnabledAndConfigured: false,
        isWindowsMdmEnabledAndConfigured: false,
        isAndroidMdmEnabledAndConfigured: false,
      });

      const ids = items.map((i) => i.id);
      expect(ids).toContain("turn-on-apple-mdm");
      expect(ids).toContain("turn-on-windows-mdm");
      expect(ids).toContain("turn-on-android-mdm");
      expect(ids).not.toContain("edit-apple-mdm");
      expect(ids).not.toContain("edit-windows-mdm");
    });

    it("shows 'Edit' MDM when configured", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        isMacMdmEnabledAndConfigured: true,
        isWindowsMdmEnabledAndConfigured: true,
        isAndroidMdmEnabledAndConfigured: true,
      });

      const ids = items.map((i) => i.id);
      expect(ids).toContain("edit-apple-mdm");
      expect(ids).toContain("edit-windows-mdm");
      expect(ids).toContain("edit-android-mdm");
      expect(ids).not.toContain("turn-on-apple-mdm");
    });

    it("shows 'Add ABM' when Apple MDM on but ABM not configured", () => {
      const configNoAbm = createMockConfig();
      configNoAbm.mdm = {
        ...configNoAbm.mdm,
        apple_bm_enabled_and_configured: false,
      };

      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        config: configNoAbm,
      });

      const abm = items.find((i) => i.id === "add-abm");
      expect(abm).toBeDefined();
      expect(abm?.label).toContain("Add");
    });

    it("shows 'Edit ABM' when ABM is configured", () => {
      const items = buildPaletteItems(BASE_CONTEXT);

      const abm = items.find((i) => i.id === "edit-abm");
      expect(abm).toBeDefined();
      expect(abm?.label).toContain("Edit");
    });

    it("shows 'Edit VPP' when VPP is enabled", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        isVppEnabled: true,
      });

      const vpp = items.find((i) => i.id === "edit-vpp");
      expect(vpp).toBeDefined();
      expect(vpp?.label).toContain("Edit");
    });

    it("shows team-scoped policy automations when premium and team selected", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: true,
        currentTeam: { id: 1, name: "Engineering" },
      });

      const policyAutomations = items.find(
        (i) => i.id === "manage-policy-automations"
      );
      expect(policyAutomations?.subItems?.length).toBeGreaterThan(1);

      const subIds = policyAutomations?.subItems?.map((s) => s.id) ?? [];
      expect(subIds).toContain("manage-policy-automations-install-software");
      expect(subIds).toContain("manage-policy-automations-calendar");
    });

    it("excludes team-scoped policy automations when no team selected", () => {
      const items = buildPaletteItems(BASE_CONTEXT);

      const policyAutomations = items.find(
        (i) => i.id === "manage-policy-automations"
      );
      // Only webhooks should be present (no team-scoped items)
      expect(policyAutomations?.subItems?.length).toBe(1);
      expect(policyAutomations?.subItems?.[0].id).toBe(
        "manage-policy-automations-webhooks"
      );
    });

    it("excludes certificates and passwords for technicians", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        isTechnician: true,
        hasTeamSelected: true,
        currentTeam: { id: 1, name: "Engineering" },
      });

      const osSettings = items.find((i) => i.id === "controls-os-settings");
      const subIds = osSettings?.subItems?.map((s) => s.id) ?? [];
      expect(subIds).not.toContain("controls-certificates");
      expect(subIds).not.toContain("controls-passwords");
      expect(subIds).toContain("controls-disk-encryption");
    });

    it("includes certificates and passwords for non-technicians", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: true,
        currentTeam: { id: 1, name: "Engineering" },
      });

      const osSettings = items.find((i) => i.id === "controls-os-settings");
      const subIds = osSettings?.subItems?.map((s) => s.id) ?? [];
      expect(subIds).toContain("controls-certificates");
      expect(subIds).toContain("controls-passwords");
    });

    it("appends fleet_id via withTeamId for team-scoped paths", () => {
      const mockWithTeamId = (path: string) => `${path}?fleet_id=5`;

      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        withTeamId: mockWithTeamId,
        hasTeamSelected: true,
        currentTeam: { id: 5, name: "Eng" },
      });

      const dashboard = items.find((i) => i.id === "dashboard");
      expect(dashboard?.path).toContain("fleet_id=5");
    });

    it("includes manage software automations without a teamName chip on All fleets", () => {
      // Only visible on All fleets, destination is All fleets — no switch,
      // no chip.
      const items = buildPaletteItems(BASE_CONTEXT);

      const swAuto = items.find((i) => i.id === "manage-software-automations");
      expect(swAuto).toBeDefined();
      expect(swAuto?.teamName).toBeUndefined();
    });

    it("calls onToggleDarkMode for the dark mode item", () => {
      const mockToggle = jest.fn();
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        onToggleDarkMode: mockToggle,
      });

      const darkMode = items.find((i) => i.id === "toggle-dark-mode");
      expect(darkMode).toBeDefined();
      darkMode?.onAction?.();
      expect(mockToggle).toHaveBeenCalled();
    });

    it("toggle-dark-mode label reflects the isDarkMode context flag", () => {
      const lightItems = buildPaletteItems({
        ...BASE_CONTEXT,
        isDarkMode: false,
      });
      expect(lightItems.find((i) => i.id === "toggle-dark-mode")?.label).toBe(
        "Switch to dark mode"
      );

      const darkItems = buildPaletteItems({
        ...BASE_CONTEXT,
        isDarkMode: true,
      });
      expect(darkItems.find((i) => i.id === "toggle-dark-mode")?.label).toBe(
        "Switch to light mode"
      );
    });

    it("hides software add, script, and variable actions on All fleets", () => {
      const items = buildPaletteItems(BASE_CONTEXT);
      const ids = items.map((i) => i.id);

      expect(ids).not.toContain("add-fleet-maintained-app");
      expect(ids).not.toContain("add-vpp-app");
      expect(ids).not.toContain("add-android-app-store-app");
      expect(ids).not.toContain("add-custom-package");
      expect(ids).not.toContain("add-script");
      expect(ids).not.toContain("add-custom-variable");
    });

    it("shows software add, script, and variable actions on Unassigned", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: false,
        currentTeam: { id: 0, name: "No team" },
      });
      const ids = items.map((i) => i.id);

      expect(ids).toContain("add-fleet-maintained-app");
      expect(ids).toContain("add-vpp-app");
      expect(ids).toContain("add-android-app-store-app");
      expect(ids).toContain("add-custom-package");
      expect(ids).toContain("add-script");
      expect(ids).toContain("add-custom-variable");
    });

    it("hides the Users settings item for non-admins", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        canAccessSettings: false,
        canManageSoftwareAutomations: false,
      });

      // `settings-users` lives in the Settings group and is gated on
      // canAccessSettings. (The old `users-page` Pages-group entry was
      // removed because it pointed to the same destination.)
      expect(items.map((i) => i.id)).not.toContain("settings-users");
    });

    it("shows Software library on Unassigned but not All fleets", () => {
      const allFleetsItems = buildPaletteItems(BASE_CONTEXT);
      expect(allFleetsItems.map((i) => i.id)).not.toContain("software-library");

      const unassignedItems = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: false,
        currentTeam: { id: 0, name: "No team" },
      });
      expect(unassignedItems.map((i) => i.id)).toContain("software-library");
    });

    it("includes Run live report and Run live policy for writers", () => {
      const items = buildPaletteItems(BASE_CONTEXT);
      const ids = items.map((i) => i.id);

      expect(ids).toContain("run-live-report");
      expect(ids).toContain("run-live-policy");
    });

    it("excludes Run live report and Run live policy for observers", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        canWrite: false,
        canRunLiveReport: false,
        canAccessControls: false,
        canAccessSettings: false,
        canManagePolicyAutomations: false,
        canManageSoftwareAutomations: false,
      });

      const ids = items.map((i) => i.id);
      expect(ids).not.toContain("run-live-report");
      expect(ids).not.toContain("run-live-policy");
    });

    it("shows Create fleet only for admins", () => {
      const adminItems = buildPaletteItems(BASE_CONTEXT);
      expect(adminItems.map((i) => i.id)).toContain("create-fleet");

      const nonAdminItems = buildPaletteItems({
        ...BASE_CONTEXT,
        canAccessSettings: false,
        canManageSoftwareAutomations: false,
      });
      expect(nonAdminItems.map((i) => i.id)).not.toContain("create-fleet");
    });
  });

  describe("Fleet Free (isPremiumTier: false)", () => {
    const FREE_CONTEXT = {
      ...BASE_CONTEXT,
      isPremiumTier: false,
      // Free has a single implicit fleet; mirror what AppContext would set.
      hasTeamSelected: true as const,
      currentTeam: { id: 1, name: "Engineering" },
    };

    it("hides all software-add commands", () => {
      const ids = buildPaletteItems(FREE_CONTEXT).map((i) => i.id);
      expect(ids).not.toContain("add-fleet-maintained-app");
      expect(ids).not.toContain("add-vpp-app");
      expect(ids).not.toContain("add-android-app-store-app");
      expect(ids).not.toContain("add-custom-package");
    });

    it("hides the Setup Experience parent and all its sub-items", () => {
      const ids = buildPaletteItems(FREE_CONTEXT).map((i) => i.id);
      expect(ids).not.toContain("controls-setup-experience");
      // Sub-items live under controls-setup-experience.subItems; absence
      // of the parent is sufficient.
    });

    it("hides Disk encryption, Certificates, and Passwords OS-settings sub-items", () => {
      const osSettings = buildPaletteItems(FREE_CONTEXT).find(
        (i) => i.id === "controls-os-settings"
      );
      const subIds = osSettings?.subItems?.map((s) => s.id) ?? [];
      expect(subIds).not.toContain("controls-disk-encryption");
      expect(subIds).not.toContain("controls-certificates");
      expect(subIds).not.toContain("controls-passwords");
      // Configuration profiles is not premium-gated; keep it.
      expect(subIds).toContain("controls-custom-settings");
    });

    it("hides MDM ABM and VPP commands", () => {
      const ids = buildPaletteItems(FREE_CONTEXT).map((i) => i.id);
      expect(ids).not.toContain("add-abm");
      expect(ids).not.toContain("edit-abm");
      expect(ids).not.toContain("add-vpp");
      expect(ids).not.toContain("edit-vpp");
    });

    it("hides premium integrations settings sub-items", () => {
      const integrations = buildPaletteItems(FREE_CONTEXT).find(
        (i) => i.id === "settings-integrations"
      );
      const subIds = integrations?.subItems?.map((s) => s.id) ?? [];
      expect(subIds).not.toContain("settings-int-calendars");
      expect(subIds).not.toContain("settings-int-change-management");
      expect(subIds).not.toContain("settings-int-certificate-authorities");
      expect(subIds).not.toContain("add-certificate-authority");
      expect(subIds).not.toContain("settings-int-conditional-access");
    });

    it("hides settings-fleets, create-fleet, and view-software-library", () => {
      const ids = buildPaletteItems(FREE_CONTEXT).map((i) => i.id);
      expect(ids).not.toContain("settings-fleets");
      expect(ids).not.toContain("create-fleet");
      expect(ids).not.toContain("view-software-library");
    });
  });

  describe("Primo Mode (isPrimoMode: true)", () => {
    const PRIMO_CONTEXT = {
      ...BASE_CONTEXT,
      isPrimoMode: true,
      hasTeamSelected: true as const,
      currentTeam: { id: 7, name: "Default" },
    };

    it("hides settings-fleets and create-fleet", () => {
      const ids = buildPaletteItems(PRIMO_CONTEXT).map((i) => i.id);
      expect(ids).not.toContain("settings-fleets");
      expect(ids).not.toContain("create-fleet");
    });

    it("shows manage-software-automations (Primo treats single fleet as all)", () => {
      const ids = buildPaletteItems(PRIMO_CONTEXT).map((i) => i.id);
      expect(ids).toContain("manage-software-automations");
    });

    it("suppresses all teamName chips because Primo never switches fleet context", () => {
      const items = buildPaletteItems(PRIMO_CONTEXT);
      const itemsWithChips = items.filter((i) => i.teamName !== undefined);
      expect(itemsWithChips).toHaveLength(0);
    });

    it("does not synthesize an 'Unassigned' or 'All fleets' chip on add-hosts", () => {
      const addHosts = buildPaletteItems(PRIMO_CONTEXT).find(
        (i) => i.id === "add-hosts"
      );
      expect(addHosts?.teamName).toBeUndefined();
    });
  });

  describe("GitOps Mode", () => {
    it("hides create-fleet when GitOps mode is configured", () => {
      const gitopsConfig = createMockConfig();
      gitopsConfig.gitops = {
        ...gitopsConfig.gitops,
        gitops_mode_enabled: true,
        repository_url: "https://github.com/fleetdm/fleet-config",
      };

      const ids = buildPaletteItems({
        ...BASE_CONTEXT,
        config: gitopsConfig,
      }).map((i) => i.id);

      expect(ids).not.toContain("create-fleet");
    });
  });
});
