import createMockConfig from "__mocks__/configMock";

import { buildCommandItems, GROUPS, ICommandPaletteContext } from "./helpers";

const BASE_CONTEXT: ICommandPaletteContext = {
  search: "",
  currentTeam: undefined,
  config: createMockConfig(),
  canAccessControls: true,
  canWrite: true,
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
        "Actions",
      ]);
    });
  });

  describe("buildCommandItems", () => {
    it("returns items for a global admin with a team selected", () => {
      const items = buildCommandItems({
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
      const items = buildCommandItems(BASE_CONTEXT);
      const ids = items.map((i) => i.id);
      expect(ids).not.toContain("controls-page");
      expect(ids).not.toContain("controls-os-updates");
      expect(ids).not.toContain("controls-os-settings");
    });

    it("excludes controls for observers", () => {
      const items = buildCommandItems({
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
      const items = buildCommandItems({
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
      const itemsNoSearch = buildCommandItems(BASE_CONTEXT);
      expect(itemsNoSearch.map((i) => i.id)).not.toContain("packs");

      const itemsWithSearch = buildCommandItems({
        ...BASE_CONTEXT,
        search: "packs",
      });
      const ids = itemsWithSearch.map((i) => i.id);
      expect(ids).toContain("packs");
      expect(ids).toContain("new-pack");
    });

    it("does not show packs when searching for 'package'", () => {
      const items = buildCommandItems({
        ...BASE_CONTEXT,
        search: "package",
      });
      expect(items.map((i) => i.id)).not.toContain("packs");
    });

    it("shows team name on team-scoped actions when a team is selected", () => {
      const items = buildCommandItems({
        ...BASE_CONTEXT,
        hasTeamSelected: true,
        currentTeam: { id: 1, name: "Engineering" },
      });

      const addHosts = items.find((i) => i.id === "add-hosts");
      expect(addHosts?.teamName).toBe("Engineering");

      const addReport = items.find((i) => i.id === "add-report");
      expect(addReport?.teamName).toBe("Engineering");
    });

    it("shows 'Unassigned' on add-hosts and manage-enroll-secrets when no team is selected", () => {
      const items = buildCommandItems(BASE_CONTEXT);

      const addHosts = items.find((i) => i.id === "add-hosts");
      expect(addHosts?.teamName).toBe("Unassigned");

      const enrollSecrets = items.find((i) => i.id === "manage-enroll-secrets");
      expect(enrollSecrets?.teamName).toBe("Unassigned");
    });

    it("shows 'All fleets' on non-host actions when no team is selected", () => {
      const items = buildCommandItems(BASE_CONTEXT);

      const addReport = items.find((i) => i.id === "add-report");
      expect(addReport?.teamName).toBe("All fleets");
    });

    it("shows 'Unassigned' when unassigned team is selected", () => {
      const items = buildCommandItems({
        ...BASE_CONTEXT,
        hasTeamSelected: false,
        currentTeam: { id: 0, name: "No team" },
      });

      const addHosts = items.find((i) => i.id === "add-hosts");
      expect(addHosts?.teamName).toBe("Unassigned");
    });

    it("shows 'Turn on' MDM when not configured", () => {
      const items = buildCommandItems({
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
      const items = buildCommandItems({
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

      const items = buildCommandItems({
        ...BASE_CONTEXT,
        config: configNoAbm,
      });

      const abm = items.find((i) => i.id === "add-abm");
      expect(abm).toBeDefined();
      expect(abm?.label).toContain("Add");
    });

    it("shows 'Edit ABM' when ABM is configured", () => {
      const items = buildCommandItems(BASE_CONTEXT);

      const abm = items.find((i) => i.id === "edit-abm");
      expect(abm).toBeDefined();
      expect(abm?.label).toContain("Edit");
    });

    it("shows 'Edit VPP' when VPP is enabled", () => {
      const items = buildCommandItems({
        ...BASE_CONTEXT,
        isVppEnabled: true,
      });

      const vpp = items.find((i) => i.id === "edit-vpp");
      expect(vpp).toBeDefined();
      expect(vpp?.label).toContain("Edit");
    });

    it("shows team-scoped policy automations when premium and team selected", () => {
      const items = buildCommandItems({
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
      const items = buildCommandItems(BASE_CONTEXT);

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
      const items = buildCommandItems({
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
      const items = buildCommandItems({
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

      const items = buildCommandItems({
        ...BASE_CONTEXT,
        withTeamId: mockWithTeamId,
        hasTeamSelected: true,
        currentTeam: { id: 5, name: "Eng" },
      });

      const dashboard = items.find((i) => i.id === "dashboard");
      expect(dashboard?.path).toContain("fleet_id=5");
    });

    it("includes manage software automations with 'All fleets' teamName", () => {
      const items = buildCommandItems(BASE_CONTEXT);

      const swAuto = items.find((i) => i.id === "manage-software-automations");
      expect(swAuto).toBeDefined();
      expect(swAuto?.teamName).toBe("All fleets");
    });

    it("calls onToggleDarkMode for the dark mode item", () => {
      const mockToggle = jest.fn();
      const items = buildCommandItems({
        ...BASE_CONTEXT,
        onToggleDarkMode: mockToggle,
      });

      const darkMode = items.find((i) => i.id === "toggle-dark-mode");
      expect(darkMode).toBeDefined();
      darkMode?.onAction?.();
      expect(mockToggle).toHaveBeenCalled();
    });

    it("hides software add, script, and variable actions on All fleets", () => {
      const items = buildCommandItems(BASE_CONTEXT);
      const ids = items.map((i) => i.id);

      expect(ids).not.toContain("add-fleet-maintained-app");
      expect(ids).not.toContain("add-vpp-app");
      expect(ids).not.toContain("add-android-app-store-app");
      expect(ids).not.toContain("add-custom-package");
      expect(ids).not.toContain("add-script");
      expect(ids).not.toContain("add-custom-variable");
    });

    it("shows software add, script, and variable actions on Unassigned", () => {
      const items = buildCommandItems({
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

    it("hides Users page for non-admins", () => {
      const items = buildCommandItems({
        ...BASE_CONTEXT,
        canAccessSettings: false,
        canManageSoftwareAutomations: false,
      });

      expect(items.map((i) => i.id)).not.toContain("users-page");
    });

    it("shows Software library on Unassigned but not All fleets", () => {
      const allFleetsItems = buildCommandItems(BASE_CONTEXT);
      expect(allFleetsItems.map((i) => i.id)).not.toContain("software-library");

      const unassignedItems = buildCommandItems({
        ...BASE_CONTEXT,
        hasTeamSelected: false,
        currentTeam: { id: 0, name: "No team" },
      });
      expect(unassignedItems.map((i) => i.id)).toContain("software-library");
    });

    it("includes Run live report and Run live policy for writers", () => {
      const items = buildCommandItems(BASE_CONTEXT);
      const ids = items.map((i) => i.id);

      expect(ids).toContain("run-live-report");
      expect(ids).toContain("run-live-policy");
    });

    it("excludes Run live report and Run live policy for observers", () => {
      const items = buildCommandItems({
        ...BASE_CONTEXT,
        canWrite: false,
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
      const adminItems = buildCommandItems(BASE_CONTEXT);
      expect(adminItems.map((i) => i.id)).toContain("create-fleet");

      const nonAdminItems = buildCommandItems({
        ...BASE_CONTEXT,
        canAccessSettings: false,
        canManageSoftwareAutomations: false,
      });
      expect(nonAdminItems.map((i) => i.id)).not.toContain("create-fleet");
    });
  });
});
