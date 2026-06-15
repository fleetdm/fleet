import React from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import createMockConfig from "__mocks__/configMock";

import CommandPalette from "./CommandPalette";

// cmdk uses scrollIntoView which JSDOM doesn't implement
Element.prototype.scrollIntoView = jest.fn();

// CommandPalette branches on navigator.platform to pick the modifier
// (Cmd on macOS, Ctrl elsewhere). jsdom's default is an empty string,
// which would make the whole suite run as "non-Mac" and break every
// {Meta>}…{/Meta} keyboard test. Default the suite to Mac and override
// per-test when we specifically want to exercise the non-Mac branch.
const setPlatform = (value: string) => {
  Object.defineProperty(window.navigator, "platform", {
    value,
    configurable: true,
  });
};
// Capture the original descriptor so afterEach can fully restore it —
// not just the value. Otherwise our `configurable: true` override leaks
// into other tests and can mask future descriptor-sensitive bugs.
const originalPlatformDescriptor = Object.getOwnPropertyDescriptor(
  window.navigator,
  "platform"
);
beforeEach(() => setPlatform("MacIntel"));
afterEach(() => {
  if (originalPlatformDescriptor) {
    Object.defineProperty(
      window.navigator,
      "platform",
      originalPlatformDescriptor
    );
  } else {
    delete (window.navigator as { platform?: string }).platform;
  }
});

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

    it("Ctrl+K does NOT open the palette on macOS (Cmd is required)", async () => {
      // Ctrl+K is readline kill-line on macOS — we must not hijack it.
      const { user } = adminRender(<CommandPalette />);
      await user.keyboard("{Control>}k{/Control}");

      expect(screen.queryByPlaceholderText(/search/i)).not.toBeInTheDocument();
    });

    it("Ctrl+K opens the palette on non-macOS platforms", async () => {
      setPlatform("Win32");
      const { user } = adminRender(<CommandPalette />);
      await user.keyboard("{Control>}k{/Control}");

      await waitFor(() => {
        expect(screen.getByPlaceholderText(/search/i)).toBeInTheDocument();
      });
    });

    it("Cmd+K does NOT open the palette on non-macOS platforms", async () => {
      setPlatform("Win32");
      const { user } = adminRender(<CommandPalette />);
      await user.keyboard("{Meta>}k{/Meta}");

      expect(screen.queryByPlaceholderText(/search/i)).not.toBeInTheDocument();
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

  describe("Keyboard shortcuts", () => {
    it("opens the switch-fleet picker page on Cmd+Shift+F", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      await user.keyboard("{Meta>}{Shift>}f{/Shift}{/Meta}");

      await waitFor(() => {
        expect(
          screen.getByPlaceholderText("Search a fleet...")
        ).toBeInTheDocument();
      });
    });

    it("Cmd+Shift+F also opens the palette directly to switch-fleet from closed", async () => {
      const { user } = adminRender(<CommandPalette />);
      // Don't openPalette first — verify cold-start behavior.
      await user.keyboard("{Meta>}{Shift>}f{/Shift}{/Meta}");

      await waitFor(() => {
        expect(
          screen.getByPlaceholderText("Search a fleet...")
        ).toBeInTheDocument();
      });
    });

    it("Ctrl+Shift+F does NOT open switch-fleet on macOS (Cmd is required)", async () => {
      const { user } = adminRender(<CommandPalette />);
      await user.keyboard("{Control>}{Shift>}f{/Shift}{/Control}");

      expect(
        screen.queryByPlaceholderText("Search a fleet...")
      ).not.toBeInTheDocument();
    });

    it("Ctrl+Shift+F opens switch-fleet on non-macOS platforms", async () => {
      setPlatform("Win32");
      const { user } = adminRender(<CommandPalette />);
      await user.keyboard("{Control>}{Shift>}f{/Shift}{/Control}");

      await waitFor(() => {
        expect(
          screen.getByPlaceholderText("Search a fleet...")
        ).toBeInTheDocument();
      });
    });

    it("Cmd+Shift+F does NOT open switch-fleet on non-macOS platforms", async () => {
      setPlatform("Win32");
      const { user } = adminRender(<CommandPalette />);
      await user.keyboard("{Meta>}{Shift>}f{/Shift}{/Meta}");

      expect(
        screen.queryByPlaceholderText("Search a fleet...")
      ).not.toBeInTheDocument();
    });

    it("Escape returns to root from a picker page instead of closing", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      // Navigate into the switch-fleet picker page
      await user.keyboard("{Meta>}{Shift>}f{/Shift}{/Meta}");
      await waitFor(() => {
        expect(
          screen.getByPlaceholderText("Search a fleet...")
        ).toBeInTheDocument();
      });

      // ESC should take us back to root, not close the dialog
      await user.keyboard("{Escape}");
      await waitFor(() => {
        expect(
          screen.getByPlaceholderText("Search for a page or command...")
        ).toBeInTheDocument();
      });
    });

    it("Escape returns to root from a picker page (view-host)", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      // The root page lists commands; find "View host" and activate it
      // to reach the view-host picker page.
      const viewHost = await screen.findByText("View host");
      await user.click(viewHost);

      await waitFor(() => {
        expect(
          screen.getByPlaceholderText("Search hosts...")
        ).toBeInTheDocument();
      });

      // ESC takes us back to root, NOT closing the dialog.
      await user.keyboard("{Escape}");
      await waitFor(() => {
        expect(
          screen.getByPlaceholderText("Search for a page or command...")
        ).toBeInTheDocument();
      });
    });

    it("Escape closes the palette when on the root page", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      await user.keyboard("{Escape}");
      await waitFor(() => {
        expect(
          screen.queryByPlaceholderText(/search/i)
        ).not.toBeInTheDocument();
      });
    });

    it("Backspace on empty input goes back from a picker page", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);
      await user.keyboard("{Meta>}{Shift>}f{/Shift}{/Meta}");
      await waitFor(() => {
        expect(
          screen.getByPlaceholderText("Search a fleet...")
        ).toBeInTheDocument();
      });

      // Backspace with empty input → root page
      await user.keyboard("{Backspace}");
      await waitFor(() => {
        expect(
          screen.getByPlaceholderText("Search for a page or command...")
        ).toBeInTheDocument();
      });
    });
  });

  describe("Sign out", () => {
    it("renders Sign out under the Commands group", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      expect(screen.getByText("Sign out")).toBeInTheDocument();
    });
  });

  describe("Dark mode reactivity", () => {
    it("updates the toggle-dark-mode label on fleet-theme-change events", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      // Initial render — theme defaults to light in tests.
      expect(screen.getByText("Switch to dark mode")).toBeInTheDocument();

      // Simulate the theme flipping to dark from elsewhere (system theme,
      // another tab, sibling component).
      window.dispatchEvent(
        new CustomEvent("fleet-theme-change", { detail: { dark: true } })
      );

      await waitFor(() => {
        expect(screen.getByText("Switch to light mode")).toBeInTheDocument();
      });
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

    it("auto-expands sub-items when a parent is highlighted via arrow keys", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      // Sub-items hidden initially.
      expect(screen.queryByText("Disk encryption")).not.toBeInTheDocument();

      // Arrow down until OS settings is highlighted (aria-selected="true").
      // Derive the upper bound from the number of rendered items so this
      // doesn't silently miss if the list grows or reorders.
      const itemCount = document.querySelectorAll(`.command-palette__item`)
        .length;
      const maxPresses = itemCount + 2;
      let osSettingsItem: Element | null = null;
      for (let i = 0; i < maxPresses; i += 1) {
        osSettingsItem = screen
          .getByText("OS settings")
          .closest(`.command-palette__item`);
        if (osSettingsItem?.getAttribute("aria-selected") === "true") {
          break;
        }
        // eslint-disable-next-line no-await-in-loop
        await user.keyboard("{ArrowDown}");
      }

      // Assert OS settings actually got selected so a failure here points
      // at the navigation step, not at the expansion check below.
      expect(osSettingsItem?.getAttribute("aria-selected")).toBe("true");

      await waitFor(() => {
        expect(screen.getByText("Disk encryption")).toBeInTheDocument();
      });
    });

    it("does not auto-expand sub-items when a parent is hovered with the mouse", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      const osSettingsItem = screen
        .getByText("OS settings")
        .closest(`.command-palette__item`);
      expect(osSettingsItem).toBeInTheDocument();

      // Hovering moves cmdk's selected value (selection-follows-pointer)
      // but must not pop sub-items open — that should only happen on
      // keyboard nav. The expand/collapse bridge runs through a
      // useEffect, so wrap the negative assertion in waitFor to make
      // sure pending effects have flushed before we conclude that
      // nothing expanded.
      await user.hover(osSettingsItem!);

      await waitFor(() => {
        expect(screen.queryByText("Disk encryption")).not.toBeInTheDocument();
      });
    });

    it("expands sub-items on chevron click", async () => {
      const { user } = adminRender(<CommandPalette />);
      await openPalette(user);

      // Sub-items should not be visible initially
      expect(screen.queryByText("Disk encryption")).not.toBeInTheDocument();

      // fireEvent here, not user.click — cmdk's userEvent-aware
      // selection handlers fire onSelect on the parent Command.Item
      // and navigate before the chevron's own click handler runs.
      // fireEvent.click dispatches a bare click that respects the
      // chevron's stopPropagation. (See cmdk + @testing-library/user-event
      // v14 compatibility.)
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

    it("does not intercept Cmd+K for isNoAccess users", async () => {
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

      const { user } = noAccessRender(<CommandPalette />);
      const onKeyDown = jest.fn();
      document.addEventListener("keydown", onKeyDown);

      await user.keyboard("{Meta>}k{/Meta}");

      // If the palette had registered its handler, it would have called
      // preventDefault on the synthetic event before our listener saw it.
      expect(onKeyDown).toHaveBeenCalled();
      expect(onKeyDown.mock.calls[0][0].defaultPrevented).toBe(false);
      // And the palette must remain unrendered.
      expect(screen.queryByPlaceholderText(/search/i)).not.toBeInTheDocument();

      document.removeEventListener("keydown", onKeyDown);
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
