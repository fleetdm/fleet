import { screen, waitFor } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import createMockUser from "__mocks__/userMock";
import { ITeamSummary } from "interfaces/team";
import { IUser } from "interfaces/user";
import PATHS from "router/paths";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import SoftwarePage, {
  softwareSubNav,
  premiumSoftwareSubNav,
  getTabIndex,
  getOSTabSortHeader,
} from "./SoftwarePage";

// These are not exported by default — we'll test the logic via the exported
// nav arrays and getTabIndex. If they aren't exported yet, see note below.

describe("SoftwarePage tab configuration", () => {
  describe("softwareSubNav (free tier)", () => {
    it("includes Inventory, OS, and Vulnerabilities tabs", () => {
      const names = softwareSubNav.map((item) => item.name);
      expect(names).toEqual(["Inventory", "OS", "Vulnerabilities"]);
    });

    it("does not include Library tab", () => {
      const names = softwareSubNav.map((item) => item.name);
      expect(names).not.toContain("Library");
    });

    it("points Inventory to SOFTWARE_INVENTORY path", () => {
      const inventory = softwareSubNav.find(
        (item) => item.name === "Inventory"
      );
      expect(inventory?.pathname).toBe(PATHS.SOFTWARE_INVENTORY);
    });
  });

  describe("premiumSoftwareSubNav (premium tier)", () => {
    it("includes Inventory, OS, Vulnerabilities, and Library tabs", () => {
      const names = premiumSoftwareSubNav.map((item) => item.name);
      expect(names).toEqual(["Inventory", "OS", "Vulnerabilities", "Library"]);
    });

    it("points Library to SOFTWARE_LIBRARY path", () => {
      const library = premiumSoftwareSubNav.find(
        (item) => item.name === "Library"
      );
      expect(library?.pathname).toBe(PATHS.SOFTWARE_LIBRARY);
    });
  });

  describe("getTabIndex", () => {
    it("returns the Inventory tab index for the inventory path", () => {
      expect(getTabIndex(PATHS.SOFTWARE_INVENTORY, premiumSoftwareSubNav)).toBe(
        0
      );
    });

    it("returns the Inventory tab index for the versions path", () => {
      expect(getTabIndex(PATHS.SOFTWARE_VERSIONS, premiumSoftwareSubNav)).toBe(
        0
      );
    });

    it("returns the OS tab index for the OS path", () => {
      expect(getTabIndex(PATHS.SOFTWARE_OS, premiumSoftwareSubNav)).toBe(1);
    });

    it("returns the Vulnerabilities tab index for the vulnerabilities path", () => {
      expect(
        getTabIndex(PATHS.SOFTWARE_VULNERABILITIES, premiumSoftwareSubNav)
      ).toBe(2);
    });

    it("returns the Library tab index for the library path", () => {
      expect(getTabIndex(PATHS.SOFTWARE_LIBRARY, premiumSoftwareSubNav)).toBe(
        3
      );
    });

    it("returns -1 for an unknown path", () => {
      expect(getTabIndex("/software/unknown", premiumSoftwareSubNav)).toBe(-1);
    });
  });

  describe("getOSTabSortHeader", () => {
    it("defaults to version once a single platform is selected on the OS tab", () => {
      expect(getOSTabSortHeader(PATHS.SOFTWARE_OS, "darwin", undefined)).toBe(
        "version"
      );
    });

    it("defaults to host count on the OS tab's 'All platforms' view", () => {
      expect(getOSTabSortHeader(PATHS.SOFTWARE_OS, "all", undefined)).toBe(
        "hosts_count"
      );
    });

    it("ignores a crafted/stale order_key=version on 'All platforms', instead of sending a nonsensical cross-platform version sort to the API", () => {
      // Comparing OS versions across platforms isn't meaningful, and the
      // Version column is unclickable on this view — but this page is
      // server-driven, so a URL with both params together (typed by hand,
      // bookmarked, or restored via browser back/forward) would otherwise
      // still reach the API as a real order_key=version request.
      expect(getOSTabSortHeader(PATHS.SOFTWARE_OS, "all", "version")).toBe(
        "hosts_count"
      );
    });

    it("still honors an explicit order_key on a single platform, or on other tabs", () => {
      expect(getOSTabSortHeader(PATHS.SOFTWARE_OS, "darwin", "name")).toBe(
        "name"
      );
      expect(
        getOSTabSortHeader(PATHS.SOFTWARE_INVENTORY, "all", "version")
      ).toBe("version");
    });
  });
});

const ALL_FLEETS: ITeamSummary[] = [
  { id: -1, name: "All fleets" },
  { id: 7, name: "Workstations" },
  { id: 0, name: "Unassigned" },
];

const GLOBAL_ADMIN = { isGlobalAdmin: true, isOnGlobalTeam: true };

const FLEET_ADMIN = {
  currentUser: createMockUser({
    global_role: null,
    teams: [{ id: 7, name: "Workstations", role: "admin" }],
  }) as IUser,
};

const renderPage = ({
  pathname,
  query = {},
  app,
  children = <div />,
}: {
  pathname: string;
  query?: Record<string, string>;
  app: Record<string, unknown>;
  children?: React.ReactElement;
}) => {
  const router = createMockRouter();
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        currentUser: createMockUser(),
        config: createMockConfig(),
        availableTeams: ALL_FLEETS,
        setCurrentTeam: jest.fn(),
        ...app,
      },
    },
  });

  const params = new URLSearchParams(query).toString();
  const rendered = render(
    <SoftwarePage
      router={router}
      location={{ pathname, search: params && `?${params}`, query, hash: "" }}
    >
      {children}
    </SoftwarePage>
  );

  return { router, ...rendered };
};

describe("SoftwarePage Library tab redirect", () => {
  it.each([
    {
      name: "All fleets is selected",
      app: { isPremiumTier: true, ...GLOBAL_ADMIN },
    },
    {
      name: "the instance is Free",
      app: { isFreeTier: true, isPremiumTier: false, ...GLOBAL_ADMIN },
    },
  ])("redirects to Inventory with no fleet id when $name", async ({ app }) => {
    const { router } = renderPage({ pathname: PATHS.SOFTWARE_LIBRARY, app });

    await waitFor(() => {
      expect(router.replace).toHaveBeenCalledWith(PATHS.SOFTWARE_INVENTORY);
    });
  });

  it.each([
    {
      name: "a fleet is selected",
      app: { isPremiumTier: true, ...FLEET_ADMIN },
    },
    {
      name: "the user's fleets are still loading",
      app: { isPremiumTier: true, ...FLEET_ADMIN, availableTeams: undefined },
    },
  ])("stays on Library when $name", async ({ app }) => {
    const { router } = renderPage({
      pathname: PATHS.SOFTWARE_LIBRARY,
      query: { fleet_id: "7" },
      app,
    });

    await waitFor(() => undefined);
    expect(router.replace).not.toHaveBeenCalled();
  });
});

describe("SoftwarePage type filter", () => {
  const FiltersOpener = ({
    onAddFiltersClick,
  }: {
    onAddFiltersClick?: () => void;
  }) => (
    <button type="button" onClick={onAddFiltersClick}>
      Open filters
    </button>
  );

  const renderInventoryTab = (query: Record<string, string>) =>
    renderPage({
      pathname: PATHS.SOFTWARE_INVENTORY,
      query,
      app: { isPremiumTier: true, ...GLOBAL_ADMIN },
      children: <FiltersOpener />,
    });

  it("restores the selection from the URL and writes it back on Apply", async () => {
    const { router, user } = renderInventoryTab({
      types: "macos_app,foo,brave_extension",
    });

    await user.click(screen.getByRole("button", { name: "Open filters" }));

    expect(
      screen
        .getAllByRole("checkbox", { checked: true })
        .map((el) => el.textContent)
    ).toEqual(["Brave extension", "macOS app"]);

    await user.click(screen.getByRole("button", { name: "Apply" }));

    expect(router.replace).toHaveBeenLastCalledWith(
      expect.stringContaining("types=brave_extension%2Cmacos_app")
    );
  });

  it("writes no types param once every type is cleared", async () => {
    const { router, user } = renderInventoryTab({ types: "macos_app" });

    await user.click(screen.getByRole("button", { name: "Open filters" }));
    await user.click(screen.getByRole("checkbox", { name: "macOS app" }));
    await user.click(screen.getByRole("button", { name: "Apply" }));

    expect(router.replace).toHaveBeenLastCalledWith(
      expect.not.stringContaining("types=")
    );
  });
});

describe("SoftwarePage AI tools filter", () => {
  const FiltersProbe = ({
    onAddFiltersClick,
    filters,
  }: {
    onAddFiltersClick?: () => void;
    filters?: Record<string, unknown>;
  }) => (
    <>
      <button type="button" onClick={onAddFiltersClick}>
        Open filters
      </button>
      <output>{JSON.stringify(filters)}</output>
    </>
  );

  const renderInventoryTab = (
    query: Record<string, string>,
    app: Record<string, unknown>
  ) =>
    renderPage({
      pathname: PATHS.SOFTWARE_INVENTORY,
      query,
      app: { ...app, ...GLOBAL_ADMIN },
      children: <FiltersProbe />,
    });

  const passedFilters = () =>
    JSON.parse(screen.getByRole("status").textContent || "{}");

  it("writes ai_tool to the URL and resets the page on Apply", async () => {
    const { router, user } = renderInventoryTab(
      { page: "3" },
      { isPremiumTier: true }
    );

    await user.click(screen.getByRole("button", { name: "Open filters" }));
    await user.click(screen.getByRole("switch", { name: "AI tools" }));
    await user.click(screen.getByRole("button", { name: "Apply" }));

    expect(router.replace).toHaveBeenLastCalledWith(
      expect.stringMatching(/page=0.*ai_tool=true|ai_tool=true.*page=0/)
    );
    expect(router.replace).toHaveBeenLastCalledWith(
      expect.not.stringContaining("vulnerable=")
    );
  });

  it("restores the toggle from the URL", async () => {
    const { user } = renderInventoryTab(
      { ai_tool: "true" },
      { isPremiumTier: true }
    );

    expect(passedFilters().aiTool).toBe(true);
    await user.click(screen.getByRole("button", { name: "Open filters" }));

    expect(screen.getByRole("switch", { name: "AI tools" })).toHaveAttribute(
      "aria-checked",
      "true"
    );
    expect(
      screen.getByRole("checkbox", { name: "MCP server" })
    ).toBeInTheDocument();
  });

  it("waits for the license tier before rendering the tab", () => {
    renderInventoryTab(
      { ai_tool: "true", types: "mcp_server" },
      { isPremiumTier: undefined }
    );

    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Open filters" })
    ).not.toBeInTheDocument();
  });

  it("ignores the AI tools filter and AI types on Fleet Free", async () => {
    const { user } = renderInventoryTab(
      { ai_tool: "true", types: "mcp_server,macos_app" },
      { isPremiumTier: false, isFreeTier: true }
    );

    expect(passedFilters()).toEqual(
      expect.objectContaining({ aiTool: false, types: ["macos_app"] })
    );

    await user.click(screen.getByRole("button", { name: "Open filters" }));

    expect(
      screen.queryByRole("switch", { name: "AI tools" })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("checkbox", { name: "MCP server" })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("checkbox", { name: "AI CLI tool" })
    ).not.toBeInTheDocument();
  });
});
