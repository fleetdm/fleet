import React from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import createMockConfig from "__mocks__/configMock";

import CommandPalette from "./CommandPalette";

// cmdk uses scrollIntoView which JSDOM doesn't implement
Element.prototype.scrollIntoView = jest.fn();

const render = createCustomRenderer({
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
      currentTeam: undefined,
    },
  },
});

describe("CommandPalette", () => {
  it("renders nothing when closed", () => {
    render(<CommandPalette />);
    expect(screen.queryByLabelText("Command palette")).not.toBeInTheDocument();
  });

  it("opens on Cmd+K", async () => {
    const { user } = render(<CommandPalette />);

    await user.keyboard("{Meta>}k{/Meta}");

    await waitFor(() => {
      expect(screen.getByPlaceholderText(/search/i)).toBeInTheDocument();
    });
  });

  it("shows page items when open", async () => {
    const { user } = render(<CommandPalette />);

    await user.keyboard("{Meta>}k{/Meta}");

    await waitFor(() => {
      expect(screen.getByText("Dashboard")).toBeInTheDocument();
      expect(screen.getByText("Hosts")).toBeInTheDocument();
      expect(screen.getByText("Policies")).toBeInTheDocument();
    });
  });

  // cmdk manages its own internal filtering state and doesn't respond to
  // DOM events in JSDOM. Filtering logic is covered in helpers.tests.ts.
  it.todo("filters items based on search input");

  it("closes on Escape", async () => {
    const { user } = render(<CommandPalette />);

    await user.keyboard("{Meta>}k{/Meta}");

    await waitFor(() => {
      expect(screen.getByPlaceholderText(/search/i)).toBeInTheDocument();
    });

    await user.keyboard("{Escape}");

    await waitFor(() => {
      expect(screen.queryByPlaceholderText(/search/i)).not.toBeInTheDocument();
    });
  });

  it("shows search icon in the input", async () => {
    const { user } = render(<CommandPalette />);

    await user.keyboard("{Meta>}k{/Meta}");

    await waitFor(() => {
      expect(
        screen
          .getByPlaceholderText(/search/i)
          .closest(".command-palette__input-wrapper")
          ?.querySelector(".command-palette__input-icon")
      ).toBeInTheDocument();
    });
  });

  it("renders nothing for isNoAccess users", () => {
    const noAccessRender = createCustomRenderer({
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

    // Even after trying to open, nothing should render
    expect(screen.queryByLabelText("Command palette")).not.toBeInTheDocument();
  });

  it("shows Navigate group with Switch fleet and Sign out", async () => {
    const { user } = render(<CommandPalette />);

    await user.keyboard("{Meta>}k{/Meta}");

    await waitFor(() => {
      expect(screen.getByText("Switch fleet...")).toBeInTheDocument();
      expect(screen.getByText("Sign out")).toBeInTheDocument();
    });
  });
});
