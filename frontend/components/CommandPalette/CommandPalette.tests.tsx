import React from "react";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import createMockConfig from "__mocks__/configMock";

import CommandPalette from "./CommandPalette";

// cmdk uses scrollIntoView which JSDOM doesn't implement
Element.prototype.scrollIntoView = jest.fn();

const adminRender = createCustomRenderer({
  withBackendMock: true,
  context: {
    app: {
      currentUser: {
        id: 1,
        name: "Test User",
        email: "test@fleet.co",
        global_role: "admin",
      },
      config: createMockConfig(),
      isGlobalAdmin: true,
      isGlobalMaintainer: false,
      isAnyTeamAdmin: false,
      isAnyTeamMaintainer: false,
      isGlobalTechnician: false,
      isAnyTeamTechnician: false,
      isPremiumTier: true,
      isMacMdmEnabledAndConfigured: true,
      isWindowsMdmEnabledAndConfigured: true,
      isAndroidMdmEnabledAndConfigured: false,
      isNoAccess: false,
      isOnlyObserver: false,
      availableTeams: [
        { id: -1, name: "All fleets" },
        { id: 1, name: "Engineering" },
        { id: 2, name: "Sales" },
      ],
      currentTeam: { id: 1, name: "Engineering" },
    },
  },
});

const observerRender = createCustomRenderer({
  withBackendMock: true,
  context: {
    app: {
      currentUser: {
        id: 2,
        name: "Observer",
        email: "observer@fleet.co",
        global_role: "observer",
      },
      config: createMockConfig(),
      isGlobalAdmin: false,
      isGlobalMaintainer: false,
      isAnyTeamAdmin: false,
      isAnyTeamMaintainer: false,
      isGlobalTechnician: false,
      isAnyTeamTechnician: false,
      isPremiumTier: true,
      isMacMdmEnabledAndConfigured: true,
      isWindowsMdmEnabledAndConfigured: true,
      isAndroidMdmEnabledAndConfigured: false,
      isNoAccess: false,
      isOnlyObserver: true,
      availableTeams: [
        { id: -1, name: "All fleets" },
        { id: 1, name: "Engineering" },
      ],
      currentTeam: { id: 1, name: "Engineering" },
    },
  },
});

const openPalette = async (user: ReturnType<typeof adminRender>["user"]) => {
  await user.keyboard("{Meta>}k{/Meta}");
  await waitFor(() => {
    expect(screen.getByPlaceholderText(/search/i)).toBeInTheDocument();
  });
};

describe("CommandPalette", () => {
  describe("Opening and closing", () => {
    it("renders nothing when closed", () => {
      adminRender(<CommandPalette />);
      expect(
        screen.queryByLabelText("Command palette")
      ).not.toBeInTheDocument();
    });

    it("opens on Cmd+K", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);
      expect(screen.getByPlaceholderText(/search/i)).toBeInTheDocument();
    });

    it("closes on Escape", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      await user.keyboard("{Escape}");

      await waitFor(() => {
        expect(
          screen.queryByPlaceholderText(/search/i)
        ).not.toBeInTheDocument();
      });
    });

    it("closes when Cmd+K is pressed again", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      await user.keyboard("{Meta>}k{/Meta}");

      await waitFor(() => {
        expect(
          screen.queryByPlaceholderText(/search/i)
        ).not.toBeInTheDocument();
      });
    });

    it("resets search when reopened", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      // Type something then close
      await user.keyboard("dashboard");
      await user.keyboard("{Escape}");

      // Reopen — input should be empty
      await openPalette(user);
      expect(screen.getByPlaceholderText(/search/i)).toHaveValue("");
    });
  });

  describe("Rendering items", () => {
    it("shows page items when open", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      expect(screen.getByText("Dashboard")).toBeInTheDocument();
      expect(screen.getByText("Hosts")).toBeInTheDocument();
      expect(screen.getByText("Policies")).toBeInTheDocument();
    });

    it("shows group headings", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      expect(screen.getByText("Pages")).toBeInTheDocument();
      expect(screen.getByText("Commands")).toBeInTheDocument();
    });

    it("shows team name on team-scoped actions", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      // Multiple items should show the team name
      const teamLabels = screen.getAllByText("Engineering");
      expect(teamLabels.length).toBeGreaterThan(0);
    });

    // cmdk only renders Command.Empty when zero items match — can't trigger
    // in JSDOM since cmdk filtering doesn't respond to DOM events.
    it.todo("shows 'No results found.' for unmatched search");
  });

  describe("Fleet switcher header", () => {
    it("shows the current fleet on the header switcher button", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      const switcher = screen.getByRole("button", { name: /Engineering/ });
      expect(switcher).toBeInTheDocument();
    });

    it("navigates to the switch-fleet page when the header button is clicked", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      const switcher = screen.getByRole("button", { name: /Engineering/ });
      await user.click(switcher);

      expect(
        screen.getByPlaceholderText("Search a fleet...")
      ).toBeInTheDocument();
    });
  });

  describe("Sign out", () => {
    it("renders Sign out under the Commands group", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      expect(screen.getByText("Sign out")).toBeInTheDocument();
    });
  });

  describe("Sub-items", () => {
    it("shows chevron on items with sub-items", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      // OS settings has sub-items
      const osSettingsItem = screen
        .getByText("OS settings")
        .closest(`.command-palette__item`);
      expect(
        osSettingsItem?.querySelector(`.command-palette__item-more`)
      ).toBeInTheDocument();
    });

    it("expands sub-items on chevron click", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      // Sub-items should not be visible initially
      expect(screen.queryByText("Disk encryption")).not.toBeInTheDocument();

      // Click the chevron on OS settings — use fireEvent since cmdk
      // intercepts user.click and navigates instead of toggling
      const chevron = screen
        .getByText("OS settings")
        .closest(`.command-palette__item`)
        ?.querySelector(`.command-palette__item-more`);

      expect(chevron).toBeInTheDocument();
      fireEvent.click(chevron!);

      await waitFor(() => {
        expect(screen.getByText("Disk encryption")).toBeInTheDocument();
      });
    });
  });

  describe("Permission gating", () => {
    it("renders nothing for isNoAccess users", () => {
      const noAccessRender = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isNoAccess: true,
            currentUser: {
              id: 1,
              name: "No Access",
              email: "noaccess@fleet.co",
              global_role: null,
            },
            config: createMockConfig(),
          },
        },
      });

      noAccessRender(<CommandPalette />);
      expect(
        screen.queryByLabelText("Command palette")
      ).not.toBeInTheDocument();
    });

    it("hides Actions and Controls for observers", async () => {
      const { user } = observerRender(<CommandPalette />);
      await openPalette(user);

      // Pages should still be visible
      expect(screen.getByText("Dashboard")).toBeInTheDocument();

      // Actions and Controls should not appear
      expect(screen.queryByText("Add hosts")).not.toBeInTheDocument();
      expect(screen.queryByText("Add report")).not.toBeInTheDocument();
      expect(screen.queryByText("OS updates")).not.toBeInTheDocument();
    });

    it("hides Settings group for non-admins", async () => {
      const { user } = observerRender(<CommandPalette />);
      await openPalette(user);

      expect(
        screen.queryByText("Organization settings")
      ).not.toBeInTheDocument();
      expect(screen.queryByText("Integrations")).not.toBeInTheDocument();
    });
  });

  // cmdk manages its own internal filtering state and doesn't respond to
  // DOM events in JSDOM. Filtering logic is covered in helpers.tests.ts.
  it.todo("filters items based on search input");
});
