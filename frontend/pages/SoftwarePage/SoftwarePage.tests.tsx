import { waitFor } from "@testing-library/react";
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

const renderLibraryTab = ({
  query = {},
  app,
}: {
  query?: { fleet_id?: string };
  app: Record<string, unknown>;
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

  const search = query.fleet_id ? `?fleet_id=${query.fleet_id}` : "";
  render(
    <SoftwarePage
      router={router}
      location={{ pathname: PATHS.SOFTWARE_LIBRARY, search, query, hash: "" }}
    >
      <div />
    </SoftwarePage>
  );

  return router;
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
    const router = renderLibraryTab({ app });

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
    const router = renderLibraryTab({ query: { fleet_id: "7" }, app });

    await waitFor(() => undefined);
    expect(router.replace).not.toHaveBeenCalled();
  });
});
