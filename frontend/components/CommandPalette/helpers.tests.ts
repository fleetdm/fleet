import createMockConfig from "__mocks__/configMock";
import paths from "router/paths";

import {
  buildFleetSwitchUrl,
  buildPaletteItems,
  computeBestMatch,
  GROUPS,
  highlightMatches,
  ICommandItem,
  ICommandPaletteContext,
  pathSupportsAllFleets,
  pathSupportsUnassigned,
  scoreMatch,
  SCORE_KEYWORD_EXACT,
  SCORE_KEYWORD_PREFIX,
  SCORE_KEYWORD_WORD_PREFIX,
  SCORE_LABEL_EXACT,
  SCORE_LABEL_PREFIX,
  SCORE_LABEL_SUBSTRING,
  SCORE_LABEL_WORD_PREFIX,
} from "./helpers";

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
  canManageReportAutomations: true,
  canEditCustomVariable: true,
  canAddSoftware: true,
  isAdminOrMaintainer: true,
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

    it("keeps toggle-dark-mode and sign-out available for observers (no write)", () => {
      // Theme is a per-user UI preference exposed via My Account → Theme
      // for every signed-in user, so the palette item must survive a
      // canWrite=false context. Sign out is the other always-on item.
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        canAccessControls: false,
        canWrite: false,
        canAccessSettings: false,
        canManagePolicyAutomations: false,
        canManageSoftwareAutomations: false,
      });

      const ids = items.map((i) => i.id);
      expect(ids).toContain("toggle-dark-mode");
      expect(ids).toContain("sign-out");
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

    it("omits the chip on add-hosts and manage-enroll-secrets when on All fleets", () => {
      // These commands intentionally do NOT switch teams from All fleets —
      // the destination page reads the existing team context, which on
      // All fleets means global enroll secrets. So no chip should render.
      const items = buildPaletteItems(BASE_CONTEXT);

      const addHosts = items.find((i) => i.id === "add-hosts");
      expect(addHosts?.teamName).toBeUndefined();

      const enrollSecrets = items.find((i) => i.id === "manage-enroll-secrets");
      expect(enrollSecrets?.teamName).toBeUndefined();
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

    it("shows 'Add AB' when Apple MDM on but AB not configured", () => {
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

    it("shows 'Edit AB' when AB is configured", () => {
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

    it("exposes Manage policy automations as a flat entry that deep-links into AutomationsModal", () => {
      // Previously the palette listed Tickets & webhooks / Install
      // software / Run script / Calendar / Conditional access as
      // sub-items, each with its own ?manage_automations=<section> URL.
      // ManagePoliciesPage never parsed those params, so all five were
      // dead links. AutomationsModal also dropped its Install software
      // / Run script sections (those are per-policy now), and the modal
      // has no per-section URL trigger. The palette now mirrors the
      // reports pattern: a single entry whose path is
      // /policies?manage_automations=1, which the page reads to open
      // the modal at its single shared body.
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: true,
        currentTeam: { id: 1, name: "Engineering" },
      });

      const policyAutomations = items.find(
        (i) => i.id === "manage-policy-automations"
      );
      expect(policyAutomations).toBeDefined();
      expect(policyAutomations?.subItems).toBeUndefined();
      expect(policyAutomations?.path).toContain("manage_automations=1");
    });

    it("hides calendar + conditional-access keywords on All fleets (modal renders only Webhooks/tickets there)", () => {
      const items = buildPaletteItems(BASE_CONTEXT);
      const policyAutomations = items.find(
        (i) => i.id === "manage-policy-automations"
      );
      const keywords = policyAutomations?.keywords ?? [];
      expect(keywords).toContain("webhook");
      expect(keywords).not.toContain("calendar");
      expect(keywords).not.toContain("google calendar");
      expect(keywords).not.toContain("conditional access");
      expect(keywords).not.toContain("sso");
    });

    it("hides calendar keywords on Unassigned (Calendar section is disabled there) but keeps conditional access", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: false,
        currentTeam: { id: 0, name: "No team" },
      });
      const policyAutomations = items.find(
        (i) => i.id === "manage-policy-automations"
      );
      const keywords = policyAutomations?.keywords ?? [];
      expect(keywords).toContain("webhook");
      expect(keywords).not.toContain("calendar");
      expect(keywords).not.toContain("google calendar");
      expect(keywords).toContain("conditional access");
      expect(keywords).toContain("sso");
    });

    it("includes all keywords on a specific team", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: true,
        currentTeam: { id: 1, name: "Engineering" },
      });
      const policyAutomations = items.find(
        (i) => i.id === "manage-policy-automations"
      );
      const keywords = policyAutomations?.keywords ?? [];
      expect(keywords).toContain("webhook");
      expect(keywords).toContain("calendar");
      expect(keywords).toContain("google calendar");
      expect(keywords).toContain("conditional access");
      expect(keywords).toContain("sso");
    });

    it("excludes certificates and passwords for technicians", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        isTechnician: true,
        isAdminOrMaintainer: false,
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

    it("includes Host names for admin/maintainer on a fleet", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: true,
        currentTeam: { id: 1, name: "Engineering" },
      });

      const osSettings = items.find((i) => i.id === "controls-os-settings");
      const subIds = osSettings?.subItems?.map((s) => s.id) ?? [];
      expect(subIds).toContain("controls-host-name-template");
    });

    it("includes Host names for admin/maintainer on 'No team'", () => {
      // The template is supported for "No team" too. Unassigned satisfies
      // hasTeamOrUnassigned, so the Controls group and the Host names entry
      // both render.
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: false,
        currentTeam: { id: 0, name: "No team" },
      });

      const osSettings = items.find((i) => i.id === "controls-os-settings");
      const subIds = osSettings?.subItems?.map((s) => s.id) ?? [];
      expect(subIds).toContain("controls-host-name-template");
    });

    it("excludes Host names for technicians", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        isTechnician: true,
        isAdminOrMaintainer: false,
        hasTeamSelected: true,
        currentTeam: { id: 1, name: "Engineering" },
      });

      const osSettings = items.find((i) => i.id === "controls-os-settings");
      const subIds = osSettings?.subItems?.map((s) => s.id) ?? [];
      expect(subIds).not.toContain("controls-host-name-template");
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
      expect(ids).not.toContain("add-self-service-category");
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
      expect(ids).toContain("add-self-service-category");
      expect(ids).toContain("add-script");
      expect(ids).toContain("add-custom-variable");
    });

    it("hides 'Add script' for technicians (page-side button is disabled for them)", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: false,
        currentTeam: { id: 0, name: "No team" },
        isTechnician: true,
      });
      const ids = items.map((i) => i.id);
      expect(ids).not.toContain("add-script");
    });

    it("hides every software-add action when !canAddSoftware (current-team observer / cross-team admin / technician)", () => {
      // A user who is admin of a different team has canWrite (via
      // isAnyTeamAdmin) but isTeamAdmin(currentTeam) is false. The
      // Add software button hides on the page; the palette must too.
      // Same for technicians, who pass canWrite but never canAddSoftware.
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: false,
        currentTeam: { id: 0, name: "No team" },
        canAddSoftware: false,
      });
      const ids = items.map((i) => i.id);

      expect(ids).not.toContain("add-fleet-maintained-app");
      expect(ids).not.toContain("add-vpp-app");
      expect(ids).not.toContain("add-android-app-store-app");
      expect(ids).not.toContain("add-custom-package");
      expect(ids).not.toContain("add-self-service-category");
      // Sanity: non-software write actions still surface.
      expect(ids).toContain("add-hosts");
    });

    it("hides 'Add custom variable' for team admins/maintainers (canWrite but !canEditCustomVariable)", () => {
      // Mirrors a team-admin context: they have canWrite (so add-script,
      // add-hosts, etc. show), but the Variables page rejects them, so
      // the variable entry must not surface.
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        hasTeamSelected: false,
        currentTeam: { id: 0, name: "No team" },
        canEditCustomVariable: false,
      });
      const ids = items.map((i) => i.id);

      expect(ids).toContain("add-script");
      expect(ids).not.toContain("add-custom-variable");
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

    it("hides Manage report automations for non-admin writers (maintainers, technicians)", () => {
      // A maintainer / technician: canWrite=true but not an admin anywhere,
      // so the destination page won't expose Manage automations.
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        canWrite: true,
        canManageReportAutomations: false,
      });
      expect(items.map((i) => i.id)).not.toContain("manage-report-automations");
    });

    it("shows Manage report automations when canManageReportAutomations is true", () => {
      const items = buildPaletteItems({
        ...BASE_CONTEXT,
        canManageReportAutomations: true,
      });
      expect(items.map((i) => i.id)).toContain("manage-report-automations");
    });

    it("shows Add fleet only for admins", () => {
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
    // Mirror production state on Free: AppContext never sets currentTeam
    // (no team picker exists), so hasTeamSelected stays false. The
    // derivation treats Free as team-or-unassigned regardless.
    const FREE_CONTEXT = {
      ...BASE_CONTEXT,
      isPremiumTier: false,
      hasTeamSelected: false as const,
      currentTeam: undefined,
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

    it("hides Disk encryption, Certificates, Passwords, and Host names OS-settings sub-items", () => {
      const osSettings = buildPaletteItems(FREE_CONTEXT).find(
        (i) => i.id === "controls-os-settings"
      );
      const subIds = osSettings?.subItems?.map((s) => s.id) ?? [];
      expect(subIds).not.toContain("controls-disk-encryption");
      expect(subIds).not.toContain("controls-certificates");
      expect(subIds).not.toContain("controls-passwords");
      expect(subIds).not.toContain("controls-host-name-template");
      // Configuration profiles is not premium-gated; keep it.
      expect(subIds).toContain("controls-custom-settings");
    });

    it("hides MDM AB and VPP commands", () => {
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

    it("hides the Controls page link (lands on OS updates premium wall on Free)", () => {
      // /controls redirects to the first permitted tab, which is OS
      // updates — a premium-walled page on Free. Free users still reach
      // the tier-free Controls sub-pages via their own palette entries
      // (OS settings, Scripts, Variables).
      const ids = buildPaletteItems(FREE_CONTEXT).map((i) => i.id);
      expect(ids).not.toContain("controls-page");
    });

    it("hides OS updates, SSO end-users, and Identity provider (each renders <PremiumFeatureMessage /> on Free)", () => {
      const items = buildPaletteItems(FREE_CONTEXT);
      const ids = items.map((i) => i.id);
      expect(ids).not.toContain("controls-os-updates");
      const integrations = items.find((i) => i.id === "settings-integrations");
      const subIds = integrations?.subItems?.map((s) => s.id) ?? [];
      expect(subIds).not.toContain("settings-int-sso-end-users");
      expect(subIds).not.toContain("settings-int-identity-provider");
    });

    it("surfaces Free-available Controls items: OS settings, Scripts, Variables", () => {
      const ids = buildPaletteItems(FREE_CONTEXT).map((i) => i.id);
      expect(ids).toContain("controls-os-settings");
      expect(ids).toContain("controls-scripts");
      expect(ids).toContain("controls-variables");
    });

    it("surfaces Free-available Controls sub-items: Configuration profiles, Script library, Script batch progress", () => {
      const items = buildPaletteItems(FREE_CONTEXT);
      const osSettingsSubIds =
        items
          .find((i) => i.id === "controls-os-settings")
          ?.subItems?.map((s) => s.id) ?? [];
      expect(osSettingsSubIds).toContain("controls-custom-settings");
      const scriptsSubIds =
        items
          .find((i) => i.id === "controls-scripts")
          ?.subItems?.map((s) => s.id) ?? [];
      expect(scriptsSubIds).toContain("controls-scripts-library");
      expect(scriptsSubIds).toContain("controls-scripts-batch-progress");
    });

    it("surfaces Add script on Free (script library is Free-available)", () => {
      const ids = buildPaletteItems(FREE_CONTEXT).map((i) => i.id);
      expect(ids).toContain("add-script");
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

  describe("buildFleetSwitchUrl", () => {
    const parse = (url: string) => new URL(url, "http://localhost");

    describe("team-required page fallback", () => {
      it("redirects to /hosts/manage when switching to All from Controls", () => {
        expect(
          buildFleetSwitchUrl({
            pathname: paths.CONTROLS,
            currentSearch: "?fleet_id=1",
            fleetId: -1,
          })
        ).toBe(paths.MANAGE_HOSTS);
      });

      it("keeps fleet_id=0 on the fallback URL when switching to Unassigned from a page that doesn't support it", () => {
        // NEW_REPORT uses includeNoTeam: false — Unassigned must bounce.
        expect(
          buildFleetSwitchUrl({
            pathname: paths.NEW_REPORT,
            currentSearch: "?fleet_id=1",
            fleetId: 0,
          })
        ).toBe(`${paths.MANAGE_HOSTS}?fleet_id=0`);
      });

      it("stays on Controls when switching to Unassigned (Controls supports includeNoTeam)", () => {
        const url = parse(
          buildFleetSwitchUrl({
            pathname: paths.CONTROLS,
            currentSearch: "?fleet_id=1",
            fleetId: 0,
          })
        );
        expect(url.pathname).toBe(paths.CONTROLS);
        expect(url.searchParams.get("fleet_id")).toBe("0");
      });

      it("stays on Software Library when switching to Unassigned (Library supports includeNoTeam)", () => {
        const url = parse(
          buildFleetSwitchUrl({
            pathname: paths.SOFTWARE_LIBRARY,
            currentSearch: "?fleet_id=1&self_service=1",
            fleetId: 0,
          })
        );
        expect(url.pathname).toBe(paths.SOFTWARE_LIBRARY);
        expect(url.searchParams.get("fleet_id")).toBe("0");
      });

      it("does not trigger the fallback on a non-team-required page", () => {
        // /hosts/manage is not in TEAM_REQUIRED_PREFIXES.
        const url = parse(
          buildFleetSwitchUrl({
            pathname: paths.MANAGE_HOSTS,
            currentSearch: "?fleet_id=1",
            fleetId: -1,
          })
        );
        expect(url.pathname).toBe(paths.MANAGE_HOSTS);
        expect(url.searchParams.has("fleet_id")).toBe(false);
      });
    });

    describe("else branch strip rules", () => {
      it("sets fleet_id when switching between specific fleets", () => {
        const url = parse(
          buildFleetSwitchUrl({
            pathname: paths.MANAGE_HOSTS,
            currentSearch: "?fleet_id=1",
            fleetId: 2,
          })
        );
        expect(url.searchParams.get("fleet_id")).toBe("2");
      });

      it("removes fleet_id when switching to All fleets", () => {
        const url = parse(
          buildFleetSwitchUrl({
            pathname: paths.MANAGE_HOSTS,
            currentSearch: "?fleet_id=1",
            fleetId: -1,
          })
        );
        expect(url.searchParams.has("fleet_id")).toBe(false);
      });

      it("strips page on any fleet switch", () => {
        const url = parse(
          buildFleetSwitchUrl({
            pathname: paths.MANAGE_HOSTS,
            currentSearch: "?fleet_id=1&page=3",
            fleetId: 2,
          })
        );
        expect(url.searchParams.has("page")).toBe(false);
      });

      it("strips legacy team_id on any fleet switch", () => {
        const url = parse(
          buildFleetSwitchUrl({
            pathname: paths.MANAGE_HOSTS,
            currentSearch: "?team_id=1",
            fleetId: 2,
          })
        );
        expect(url.searchParams.has("team_id")).toBe(false);
        expect(url.searchParams.get("fleet_id")).toBe("2");
      });

      it("strips script_batch_execution_id and script_batch_execution_status on any fleet switch", () => {
        const url = parse(
          buildFleetSwitchUrl({
            pathname: paths.MANAGE_HOSTS,
            currentSearch:
              "?fleet_id=1&script_batch_execution_id=abc&script_batch_execution_status=ran",
            fleetId: 2,
          })
        );
        expect(url.searchParams.has("script_batch_execution_id")).toBe(false);
        expect(url.searchParams.has("script_batch_execution_status")).toBe(
          false
        );
      });

      it("strips software_status when switching to All fleets", () => {
        const url = parse(
          buildFleetSwitchUrl({
            pathname: paths.MANAGE_HOSTS,
            currentSearch: "?fleet_id=1&software_status=pending",
            fleetId: -1,
          })
        );
        expect(url.searchParams.has("software_status")).toBe(false);
      });

      it("preserves software_status when switching between specific fleets", () => {
        const url = parse(
          buildFleetSwitchUrl({
            pathname: paths.MANAGE_HOSTS,
            currentSearch: "?fleet_id=1&software_status=pending",
            fleetId: 2,
          })
        );
        expect(url.searchParams.get("software_status")).toBe("pending");
      });

      it("preserves unrelated params across a fleet switch", () => {
        const url = parse(
          buildFleetSwitchUrl({
            pathname: paths.MANAGE_HOSTS,
            currentSearch: "?fleet_id=1&query=foo&order_key=name",
            fleetId: 2,
          })
        );
        expect(url.searchParams.get("query")).toBe("foo");
        expect(url.searchParams.get("order_key")).toBe("name");
      });

      it("returns a bare pathname when no params remain", () => {
        expect(
          buildFleetSwitchUrl({
            pathname: paths.MANAGE_HOSTS,
            currentSearch: "?fleet_id=1",
            fleetId: -1,
          })
        ).toBe(paths.MANAGE_HOSTS);
      });
    });
  });

  describe("pathSupportsUnassigned", () => {
    it("returns true for hosts pages (manage and details)", () => {
      expect(pathSupportsUnassigned(paths.MANAGE_HOSTS)).toBe(true);
      expect(pathSupportsUnassigned(paths.HOST_DETAILS(42))).toBe(true);
    });

    it("returns true for software pages", () => {
      expect(pathSupportsUnassigned(paths.SOFTWARE)).toBe(true);
      expect(pathSupportsUnassigned(paths.SOFTWARE_TITLE_DETAILS("3"))).toBe(
        true
      );
    });

    it("returns true for controls pages", () => {
      expect(pathSupportsUnassigned(paths.CONTROLS)).toBe(true);
      expect(pathSupportsUnassigned(paths.CONTROLS_SCRIPTS)).toBe(true);
    });

    it("returns true for policies pages", () => {
      expect(pathSupportsUnassigned(paths.MANAGE_POLICIES)).toBe(true);
      expect(pathSupportsUnassigned(paths.POLICY_DETAILS(7))).toBe(true);
      expect(pathSupportsUnassigned(paths.EDIT_POLICY(7))).toBe(true);
      expect(pathSupportsUnassigned(paths.NEW_POLICY)).toBe(true);
    });

    it("returns false for Dashboard", () => {
      expect(pathSupportsUnassigned(paths.DASHBOARD)).toBe(false);
    });

    it("returns false for Reports pages", () => {
      expect(pathSupportsUnassigned(paths.MANAGE_REPORTS)).toBe(false);
      expect(pathSupportsUnassigned(paths.NEW_REPORT)).toBe(false);
    });

    it("returns false for admin/settings pages", () => {
      expect(pathSupportsUnassigned("/settings/teams/1")).toBe(false);
    });
  });

  describe("pathSupportsAllFleets", () => {
    it("returns true for top-level team-aware pages", () => {
      expect(pathSupportsAllFleets(paths.DASHBOARD)).toBe(true);
      expect(pathSupportsAllFleets(paths.MANAGE_HOSTS)).toBe(true);
      expect(pathSupportsAllFleets(paths.SOFTWARE)).toBe(true);
      expect(pathSupportsAllFleets(paths.MANAGE_POLICIES)).toBe(true);
      expect(pathSupportsAllFleets(paths.MANAGE_REPORTS)).toBe(true);
    });

    it("returns false for fleet admin detail pages (Users/Options/Settings)", () => {
      expect(pathSupportsAllFleets(paths.FLEET_DETAILS_USERS(1))).toBe(false);
      expect(pathSupportsAllFleets(paths.FLEET_DETAILS_OPTIONS(1))).toBe(false);
      expect(pathSupportsAllFleets(paths.FLEET_DETAILS_SETTINGS(1))).toBe(
        false
      );
    });

    it("still returns true for the /settings/fleets list page", () => {
      // The list view itself has no fleet scope (no useTeamIdParam), so
      // switching from the palette there is fine. Only the detail sub-pages
      // (which require a specific fleet_id) hide All.
      expect(pathSupportsAllFleets(paths.ADMIN_FLEETS)).toBe(true);
    });
  });

  describe("scoreMatch", () => {
    it("returns the label-exact tier when text equals the query (isLabel)", () => {
      expect(scoreMatch("settings", "settings", true)).toBe(SCORE_LABEL_EXACT);
    });

    it("returns the label-prefix tier on a startsWith match", () => {
      expect(scoreMatch("settings", "sett", true)).toBe(SCORE_LABEL_PREFIX);
    });

    it("returns the label-word-prefix tier when a non-first word starts with the query", () => {
      // "user settings" — query "sett" word-matches "settings" (word #2).
      expect(scoreMatch("user settings", "sett", true)).toBe(
        SCORE_LABEL_WORD_PREFIX
      );
    });

    it("treats hyphens as word boundaries for word-prefix matching", () => {
      // "API-only user" splits to ["api", "only", "user"]. Query "only"
      // word-prefix-matches the middle token. Without hyphen splitting
      // this would fall through to substring (label only) or 0 (keyword).
      expect(scoreMatch("api-only user", "only", true)).toBe(
        SCORE_LABEL_WORD_PREFIX
      );
      expect(scoreMatch("fleet-maintained app", "maintained", false)).toBe(
        SCORE_KEYWORD_WORD_PREFIX
      );
    });

    it("returns the label-substring tier on an interior substring match", () => {
      // "etting" is an interior substring of "settings" — no word prefix.
      expect(scoreMatch("settings", "etting", true)).toBe(
        SCORE_LABEL_SUBSTRING
      );
    });

    it("returns the keyword tier ladder when isLabel is false", () => {
      expect(scoreMatch("create user", "create user", false)).toBe(
        SCORE_KEYWORD_EXACT
      );
      expect(scoreMatch("create user", "create", false)).toBe(
        SCORE_KEYWORD_PREFIX
      );
      expect(scoreMatch("create user", "user", false)).toBe(
        SCORE_KEYWORD_WORD_PREFIX
      );
    });

    it("ranks any label tier above the strongest keyword tier", () => {
      // Even the weakest label hit (substring) must outrank an exact
      // keyword match — that's the load-bearing invariant.
      expect(SCORE_LABEL_SUBSTRING).toBeGreaterThan(SCORE_KEYWORD_EXACT);
    });

    it("does not return a substring tier for keywords", () => {
      // "mal" is an interior substring of keyword "amalgam" — keywords
      // skip the substring tier to keep short-keyword noise out.
      expect(scoreMatch("amalgam", "mal", false)).toBe(0);
    });

    it("returns 0 when text or query is empty", () => {
      expect(scoreMatch("", "sett", true)).toBe(0);
      expect(scoreMatch("settings", "", true)).toBe(0);
    });

    it("returns 0 when the query has no match anywhere in the text", () => {
      expect(scoreMatch("dashboard", "xyz", true)).toBe(0);
    });
  });

  describe("computeBestMatch", () => {
    const ITEM = (
      id: string,
      label: string,
      keywords?: string[]
    ): ICommandItem => ({
      id,
      label,
      group: "Pages",
      path: `/${id}`,
      keywords,
    });

    it("returns an empty array when the query is shorter than 2 chars", () => {
      const items = [ITEM("settings-page", "Settings")];
      expect(computeBestMatch(items, "")).toEqual([]);
      expect(computeBestMatch(items, "s")).toEqual([]);
    });

    it("at 2 chars, restricts to label exact + label prefix tiers", () => {
      const items = [
        // Label prefix match — qualifies at 2 chars.
        ITEM("os-settings", "OS settings"),
        // Word-prefix only — does NOT qualify at 2 chars (tier 80 < 90).
        ITEM("user-os", "User OS"),
        // Keyword match only — does NOT qualify.
        ITEM("dashboard", "Dashboard", ["os"]),
      ];
      const result = computeBestMatch(items, "os");
      expect(result.map((e) => e.item.id)).toEqual(["os-settings"]);
      expect(result[0].score).toBe(SCORE_LABEL_PREFIX);
    });

    it("at 3+ chars, unlocks word-prefix, substring, and keyword tiers", () => {
      const items = [
        ITEM("user-settings", "User settings"), // label word-prefix
        ITEM("zeta", "Zeta", ["settings cmd"]), // keyword prefix
      ];
      const result = computeBestMatch(items, "sett");
      expect(result.map((e) => e.item.id)).toEqual(["user-settings", "zeta"]);
    });

    it("gates on typed characters, not whitespace — 'o s' is treated as 2 chars", () => {
      // "o s" trims to length 3 but only 2 typed letters. Must use the
      // restricted 2-char ladder, not the full one. Without the fix, a
      // multi-token word-prefix match (score 80) would slip past the
      // floor (minScore=1) and surface — but at 2 typed chars the floor
      // is 90, so this must be rejected.
      const items = [ITEM("user-os", "User OS")];
      expect(computeBestMatch(items, "o s")).toEqual([]);

      // Same shape with 3 typed chars uses the full ladder and accepts.
      const fullResult = computeBestMatch(items, "us os");
      expect(fullResult.length).toBe(1);
    });

    it("promotes a label prefix match — 'sett' → 'Settings'", () => {
      // The motivating case from #39018 manager feedback: a partial label
      // match should land in Best match, not get jumbled below keyword
      // matches.
      const items = [
        ITEM("dashboard", "Dashboard"),
        ITEM("settings-page", "Settings", ["admin", "organization"]),
        ITEM("hosts", "Hosts"),
      ];
      const result = computeBestMatch(items, "sett");
      expect(result.map((e) => e.item.id)).toEqual(["settings-page"]);
      expect(result[0].score).toBe(SCORE_LABEL_PREFIX);
    });

    it("promotes a keyword exact match — 'create user' → 'Add user'", () => {
      // Mirrors the real "Add user" item which carries "create user"
      // as a synonym keyword. Promoted into Best match — but at the
      // keyword tier (below any label match).
      const items = [
        ITEM("add-user", "Add user", ["create user", "new user"]),
        ITEM("hosts", "Hosts"),
      ];
      const result = computeBestMatch(items, "create user");
      expect(result.map((e) => e.item.id)).toEqual(["add-user"]);
      expect(result[0].score).toBe(SCORE_KEYWORD_EXACT);
    });

    it("ranks every label match above every keyword match", () => {
      const items = [
        // Keyword exact for query "settings".
        ITEM("keyword-only", "Some action", ["settings"]),
        // Label substring (weakest label tier).
        ITEM("contains-settings", "Reset settings cache"),
      ];
      const result = computeBestMatch(items, "settings");
      // Label substring outranks keyword exact, even though both items
      // match the query.
      expect(result.map((e) => e.item.id)).toEqual([
        "contains-settings",
        "keyword-only",
      ]);
    });

    it("orders entries by score desc, then alphabetical within a tier", () => {
      const items = [
        ITEM("settings-page", "Settings"), // label exact: 100
        ITEM("settings-page-two", "Settings advanced"), // label prefix: 90
        ITEM("user-settings", "User settings"), // label word-prefix: 80
        ITEM("zeta", "Zeta", ["settings cmd"]), // keyword prefix: 40
      ];
      const result = computeBestMatch(items, "settings");
      expect(result.map((e) => e.item.id)).toEqual([
        "settings-page",
        "settings-page-two",
        "user-settings",
        "zeta",
      ]);
    });

    it("ranks an item by its strongest match across label and keywords", () => {
      // Label "Some label" doesn't match "host" at all; keyword "host"
      // does (keyword exact = 50). The item still scores at the keyword
      // tier, not zero.
      const items = [ITEM("a", "Some label", ["host"])];
      const result = computeBestMatch(items, "host");
      expect(result[0].score).toBe(SCORE_KEYWORD_EXACT);
    });

    it("does not promote items whose only hit is a keyword substring", () => {
      // Keyword "amalgam" contains "mal" — but keywords skip substring,
      // so this item must not appear.
      const items = [ITEM("a", "Unrelated", ["amalgam"]), ITEM("b", "Other")];
      expect(computeBestMatch(items, "mal")).toEqual([]);
    });

    it("promotes sub-items independently of the parent", () => {
      const parent: ICommandItem = {
        id: "controls",
        label: "Controls",
        group: "Controls",
        subItems: [
          {
            id: "controls-os-updates",
            label: "OS updates",
            path: "/controls/os-updates",
            keywords: ["patch", "minimum version"],
          },
          {
            id: "controls-passwords",
            label: "Passwords",
            path: "/controls/passwords",
          },
        ],
      };
      // "passw" matches the sub-item label "Passwords" but not the
      // parent's "Controls" label or keywords.
      const result = computeBestMatch([parent], "passw");
      expect(result).toHaveLength(1);
      expect(result[0].sub?.id).toBe("controls-passwords");
      expect(result[0].item.id).toBe("controls");
    });

    it("emits separate entries for parent and matching sub-item", () => {
      const parent: ICommandItem = {
        id: "settings-org",
        label: "Organization settings",
        group: "Settings",
        subItems: [
          {
            id: "settings-org-smtp",
            label: "SMTP options",
            path: "/settings/smtp",
            keywords: ["settings email"],
          },
        ],
      };
      // "settings" matches both:
      //  - parent label word-prefix (80)
      //  - sub-item keyword prefix (40)
      // Parent's label tier outranks the sub-item's keyword tier.
      const result = computeBestMatch([parent], "settings");
      expect(result).toHaveLength(2);
      expect(result[0].sub).toBeUndefined();
      expect(result[0].item.id).toBe("settings-org");
      expect(result[1].sub?.id).toBe("settings-org-smtp");
    });

    it("is case-insensitive on both label and query, and trims whitespace", () => {
      const items = [ITEM("a", "Settings")];
      expect(computeBestMatch(items, "SETT")[0]?.score).toBe(
        SCORE_LABEL_PREFIX
      );
      expect(computeBestMatch(items, "  Sett  ")[0]?.score).toBe(
        SCORE_LABEL_PREFIX
      );
    });

    describe("multi-token queries", () => {
      it("promotes 'Organization settings' for the order-independent query 'settings org'", () => {
        // The motivating case for multi-token: typing fragments in any
        // order should find the item. Neither "settings org" nor
        // "org settings" matches as a single phrase, but both tokens
        // find homes in the label.
        const items = [
          ITEM("settings-org", "Organization settings"),
          ITEM("unrelated", "Reports"),
        ];
        const result = computeBestMatch(items, "settings org");
        expect(result.map((e) => e.item.id)).toEqual(["settings-org"]);
        // Each token word-prefixes a label word; min = label word-prefix.
        expect(result[0].score).toBe(SCORE_LABEL_WORD_PREFIX);
      });

      it("works regardless of token order", () => {
        const items = [ITEM("settings-org", "Organization settings")];
        const a = computeBestMatch(items, "settings org");
        const b = computeBestMatch(items, "org settings");
        expect(a[0].score).toBe(b[0].score);
      });

      it("rejects items where any token has no match", () => {
        // Token "xyz" doesn't match anywhere — even though "settings"
        // matches the label, the item must not promote.
        const items = [ITEM("settings-page", "Settings")];
        expect(computeBestMatch(items, "xyz settings")).toEqual([]);
      });

      it("does not regress single-token queries", () => {
        // Single-token behavior must be unchanged by the multi-token
        // logic. 'sett' against 'Settings' is still label prefix.
        const items = [ITEM("settings-page", "Settings")];
        expect(computeBestMatch(items, "sett")[0].score).toBe(
          SCORE_LABEL_PREFIX
        );
      });

      it("preserves full-query exact match when tokens individually score lower", () => {
        // Query "create user" exact-matches the keyword "create user"
        // (keyword exact = 50). Per-token min is only 40 (token "create"
        // is keyword prefix, token "user" is label word-prefix → min 40).
        // The full-query single-pass score (50) wins.
        const items = [
          ITEM("add-user", "Add user", ["create user", "new user"]),
        ];
        const result = computeBestMatch(items, "create user");
        expect(result[0].score).toBe(SCORE_KEYWORD_EXACT);
      });

      it("matches tokens against different texts within the same item", () => {
        // Token "settings" matches the label word "settings"; token "ssa"
        // matches a keyword. Different sources, same item.
        const items = [
          ITEM("settings-org", "Organization settings", ["ssa config"]),
        ];
        const result = computeBestMatch(items, "settings ssa");
        expect(result).toHaveLength(1);
        expect(result[0].item.id).toBe("settings-org");
      });
    });
  });

  describe("highlightMatches", () => {
    it("returns a single non-matched segment when the query is empty", () => {
      expect(highlightMatches("Settings", "")).toEqual([
        { text: "Settings", matched: false },
      ]);
    });

    it("returns a single non-matched segment when nothing matches", () => {
      expect(highlightMatches("Settings", "xyz")).toEqual([
        { text: "Settings", matched: false },
      ]);
    });

    it("highlights a prefix match", () => {
      expect(highlightMatches("Settings", "Sett")).toEqual([
        { text: "Sett", matched: true },
        { text: "ings", matched: false },
      ]);
    });

    it("highlights a substring match", () => {
      expect(highlightMatches("Settings", "ting")).toEqual([
        { text: "Set", matched: false },
        { text: "ting", matched: true },
        { text: "s", matched: false },
      ]);
    });

    it("highlights every token of a multi-token query", () => {
      expect(highlightMatches("Organization settings", "org sett")).toEqual([
        { text: "Org", matched: true },
        { text: "anization ", matched: false },
        { text: "sett", matched: true },
        { text: "ings", matched: false },
      ]);
    });

    it("merges overlapping ranges from multiple tokens", () => {
      // "set" and "settings" overlap — should merge into one matched
      // range covering "settings".
      expect(highlightMatches("Settings", "set settings")).toEqual([
        { text: "Settings", matched: true },
      ]);
    });

    it("is case-insensitive but preserves the original casing in output", () => {
      // Match against "ORG" should highlight "Org" in original case.
      expect(highlightMatches("Organization settings", "ORG")).toEqual([
        { text: "Org", matched: true },
        { text: "anization settings", matched: false },
      ]);
    });

    it("handles every occurrence of a token within the text", () => {
      // Both occurrences of "se" should highlight.
      const result = highlightMatches("setup setup", "se");
      expect(result).toEqual([
        { text: "se", matched: true },
        { text: "tup ", matched: false },
        { text: "se", matched: true },
        { text: "tup", matched: false },
      ]);
    });

    it("slices the original text correctly when a match follows a length-changing case fold", () => {
      // "İ" (U+0130) lowercases to "i̇" (U+0069 + U+0307), expanding
      // 1 char into 2. A naive textLower.indexOf + text.slice would find
      // "stan" at lower-index 2 (the i + combining dot pushed it
      // forward), then slice the original at [2, 6] — producing "tanb"
      // instead of "stan". The offset map translates the lower-coord
      // range back to the original-text range.
      const result = highlightMatches("İstanbul", "stan");
      expect(result.map((s) => s.text).join("")).toBe("İstanbul");
      const matched = result.filter((s) => s.matched).map((s) => s.text);
      expect(matched).toEqual(["stan"]);
    });

    it("matches an ASCII query against a source char whose lowercase expands (İ → i̇)", () => {
      // utf8mb4_unicode_ci treats İ and i as equivalent, so the backend
      // returns "İstanbul" for query "i". The frontend must do the same
      // or rows render with zero highlights. The match in textLower
      // ("i̇stanbul") ends mid-folded-char; the offset map expands it
      // to cover the whole "İ".
      const result = highlightMatches("İstanbul", "i");
      expect(result.map((s) => s.text).join("")).toBe("İstanbul");
      expect(result[0]).toEqual({ text: "İ", matched: true });
    });

    it("matches a multi-char-lowercased query against the source char that produced it", () => {
      // Query "İ" lowercases to "i̇" (2 chars); source "İ" also lowercases
      // to "i̇". The whole source char should highlight in its original
      // form.
      const result = highlightMatches("İstanbul", "İ");
      expect(result.map((s) => s.text).join("")).toBe("İstanbul");
      expect(result[0]).toEqual({ text: "İ", matched: true });
    });

    it("folds supplementary-plane characters (Adlam)", () => {
      // U+1E900 (Adlam capital A) lowercases to U+1E922 (Adlam small a).
      // Per-code-unit folding would leave the lone surrogates unchanged
      // and silently drop the match.
      const result = highlightMatches("\u{1E900}stanbul", "\u{1E922}stanbul");
      expect(result).toEqual([{ text: "\u{1E900}stanbul", matched: true }]);
    });

    it("matches a multi-char ASCII query across a length-changing fold", () => {
      // 'İ' → 'i' + combining dot splits the run; without combining-mark
      // stripping, indexOf('istanbul') in the folded text returns -1.
      const result = highlightMatches("İstanbul", "istanbul");
      expect(result).toEqual([{ text: "İstanbul", matched: true }]);
    });

    it("matches accent-insensitively to mirror utf8mb4_unicode_ci", () => {
      // Backend returns 'Café Server' for query 'cafe'; highlighter must
      // do the same or the row renders with zero <mark> tags.
      const result = highlightMatches("Café Server", "cafe");
      expect(result).toEqual([
        { text: "Café", matched: true },
        { text: " Server", matched: false },
      ]);
    });

    it("matches when the query itself carries an accent", () => {
      const result = highlightMatches("Cafe Server", "café");
      expect(result).toEqual([
        { text: "Cafe", matched: true },
        { text: " Server", matched: false },
      ]);
    });
  });
});
