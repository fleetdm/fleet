import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { noop } from "lodash";
// TODO: Replace renderWithAppContext with createCustomRenderer
import { renderWithAppContext } from "test/test-utils";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";

import FleetsDropdown from "./FleetsDropdown";

const mockPush = jest.fn();
jest.mock("react-router", () => ({
  browserHistory: {
    push: (...args: unknown[]) => mockPush(...args),
  },
}));

// The visible trigger is a real Fleet <Button>; getByRole("button") finds it
// unambiguously (react-select's hidden control has no role="button").
const getTrigger = (name: RegExp) =>
  screen.getByRole("button", { name, hidden: false });

describe("FleetsDropdown - component", () => {
  const USER_FLEETS = [
    { id: -1, name: "All fleets" },
    { id: 1, name: "Fleet 1" },
    { id: 2, name: "Fleet 2" },
  ];

  beforeEach(() => {
    mockPush.mockClear();
  });

  it("renders the given selected fleet from selectedFleetId", () => {
    render(
      <FleetsDropdown
        currentUserFleets={USER_FLEETS}
        selectedFleetId={1}
        onChange={noop}
      />
    );

    expect(getTrigger(/Fleet 1/)).toBeInTheDocument();
  });

  it("renders the first fleet option when includeAllFleets is false and when no selectedFleetId is given", () => {
    render(
      <FleetsDropdown
        currentUserFleets={USER_FLEETS}
        includeAllFleets={false}
        onChange={noop}
      />
    );

    expect(getTrigger(/Fleet 1/)).toBeInTheDocument();
  });

  it("renders 'All fleets' when no selectedFleetId is given", () => {
    render(<FleetsDropdown currentUserFleets={USER_FLEETS} onChange={noop} />);

    expect(getTrigger(/All fleets/)).toBeInTheDocument();
  });

  it("renders the first fleet when the current-user list has no 'All fleets' row and no selectedFleetId is given", () => {
    const withoutAllFleets = USER_FLEETS.filter(
      (t) => t.id > APP_CONTEXT_NO_TEAM_ID
    );
    render(
      <FleetsDropdown currentUserFleets={withoutAllFleets} onChange={noop} />
    );

    expect(getTrigger(/Fleet 1/)).toBeInTheDocument();
  });

  describe("in-menu search", () => {
    // Search only appears once the list has 10+ rows.
    const MANY_FLEETS = [
      { id: -1, name: "All fleets" },
      { id: 1, name: "Fleet 1" },
      { id: 2, name: "Fleet 2" },
      { id: 3, name: "Fleet 3" },
      { id: 4, name: "Fleet 4" },
      { id: 5, name: "Fleet 5" },
      { id: 6, name: "Fleet 6" },
      { id: 7, name: "Fleet 7" },
      { id: 8, name: "Fleet 8" },
      { id: 9, name: "Fleet 9" },
    ];

    it("hides the search input when there are fewer than 10 rows", async () => {
      const user = userEvent.setup();
      render(
        <FleetsDropdown
          currentUserFleets={USER_FLEETS}
          selectedFleetId={1}
          onChange={noop}
        />
      );

      await user.click(getTrigger(/Fleet 1/));
      expect(
        screen.queryByPlaceholderText("Search fleets")
      ).not.toBeInTheDocument();
    });

    it("renders a search input when there are 10 or more rows", async () => {
      const user = userEvent.setup();
      render(
        <FleetsDropdown
          currentUserFleets={MANY_FLEETS}
          selectedFleetId={1}
          onChange={noop}
        />
      );

      await user.click(getTrigger(/Fleet 1/));
      expect(screen.getByPlaceholderText("Search fleets")).toBeInTheDocument();
    });

    it("filters options by the search query", async () => {
      const user = userEvent.setup();
      render(
        <FleetsDropdown
          currentUserFleets={MANY_FLEETS}
          selectedFleetId={1}
          onChange={noop}
        />
      );

      await user.click(getTrigger(/Fleet 1/));
      fireEvent.change(screen.getByPlaceholderText("Search fleets"), {
        target: { value: "Fleet 2" },
      });

      // The trigger button also contains "Fleet 1"; scope option lookups to
      // react-select's option class so the trigger doesn't count.
      const optionLabels = Array.from(
        document.querySelectorAll(".fleet-dropdown__option")
      ).map((o) => o.textContent);
      expect(optionLabels).toContain("Fleet 2");
      expect(optionLabels).not.toContain("All fleets");
    });

    it("shows the empty-state message when nothing matches", async () => {
      const user = userEvent.setup();
      render(
        <FleetsDropdown
          currentUserFleets={MANY_FLEETS}
          selectedFleetId={1}
          onChange={noop}
        />
      );

      await user.click(getTrigger(/Fleet 1/));
      fireEvent.change(screen.getByPlaceholderText("Search fleets"), {
        target: { value: "nothing-matches-this" },
      });

      expect(screen.getByText("No matching fleets")).toBeInTheDocument();
    });

    it("click outside the wrapper closes the menu and clears the search query", async () => {
      const user = userEvent.setup();
      render(
        <div>
          <button type="button">outside</button>
          <FleetsDropdown
            currentUserFleets={MANY_FLEETS}
            selectedFleetId={1}
            onChange={noop}
          />
        </div>
      );

      await user.click(getTrigger(/Fleet 1/));
      fireEvent.change(screen.getByPlaceholderText("Search fleets"), {
        target: { value: "Fleet 2" },
      });
      expect(screen.getByPlaceholderText("Search fleets")).toHaveValue(
        "Fleet 2"
      );

      // Click on an element outside the dropdown wrapper.
      fireEvent.mouseDown(screen.getByRole("button", { name: /outside/i }));

      // Menu closes.
      expect(
        screen.queryByPlaceholderText("Search fleets")
      ).not.toBeInTheDocument();

      // Reopen — the search input should be empty, not stuck on "Fleet 2".
      await user.click(getTrigger(/Fleet 1/));
      expect(screen.getByPlaceholderText("Search fleets")).toHaveValue("");
    });

    it("Escape on the search input closes the menu via the forwardNavKey bridge", async () => {
      const user = userEvent.setup();
      render(
        <FleetsDropdown
          currentUserFleets={MANY_FLEETS}
          selectedFleetId={1}
          onChange={noop}
        />
      );

      await user.click(getTrigger(/Fleet 1/));
      expect(screen.getByPlaceholderText("Search fleets")).toBeInTheDocument();

      // Escape hits the search input's onKeyDown, gets forwarded to
      // react-select's hidden input, which closes the menu. If the bridge
      // ever regresses, the search input stays mounted.
      fireEvent.keyDown(screen.getByPlaceholderText("Search fleets"), {
        key: "Escape",
      });

      expect(
        screen.queryByPlaceholderText("Search fleets")
      ).not.toBeInTheDocument();
    });
  });

  describe("Add fleet button", () => {
    const MANY_FLEETS = [
      { id: -1, name: "All fleets" },
      { id: 1, name: "Fleet 1" },
      { id: 2, name: "Fleet 2" },
      { id: 3, name: "Fleet 3" },
      { id: 4, name: "Fleet 4" },
      { id: 5, name: "Fleet 5" },
      { id: 6, name: "Fleet 6" },
      { id: 7, name: "Fleet 7" },
      { id: 8, name: "Fleet 8" },
      { id: 9, name: "Fleet 9" },
    ];

    it("does not render for non-global-admin users", async () => {
      const user = userEvent.setup();
      renderWithAppContext(
        <FleetsDropdown
          currentUserFleets={USER_FLEETS}
          selectedFleetId={1}
          onChange={noop}
        />,
        { contextValue: { isGlobalAdmin: false } }
      );

      await user.click(getTrigger(/Fleet 1/));
      expect(
        screen.queryByRole("button", { name: /add fleet/i })
      ).not.toBeInTheDocument();
    });

    it("renders as a labeled footer for global admins when the list is short", async () => {
      const user = userEvent.setup();
      renderWithAppContext(
        <FleetsDropdown
          currentUserFleets={USER_FLEETS}
          selectedFleetId={1}
          onChange={noop}
        />,
        { contextValue: { isGlobalAdmin: true } }
      );

      await user.click(getTrigger(/Fleet 1/));
      const addButton = screen.getByRole("button", { name: /add fleet/i });
      expect(addButton).toHaveTextContent("Add fleet");

      await user.click(addButton);
      expect(mockPush).toHaveBeenCalledWith("/settings/fleets?create_fleet=1");
    });

    it("renders the same labeled footer for global admins when the list is long", async () => {
      const user = userEvent.setup();
      renderWithAppContext(
        <FleetsDropdown
          currentUserFleets={MANY_FLEETS}
          selectedFleetId={1}
          onChange={noop}
        />,
        { contextValue: { isGlobalAdmin: true } }
      );

      await user.click(getTrigger(/Fleet 1/));
      const addButton = screen.getByRole("button", { name: /add fleet/i });
      expect(addButton).toHaveTextContent("Add fleet");

      await user.click(addButton);
      expect(mockPush).toHaveBeenCalledWith("/settings/fleets?create_fleet=1");
    });

    it("hides the button for global admins when GitOps mode is enabled", async () => {
      const user = userEvent.setup();
      renderWithAppContext(
        <FleetsDropdown
          currentUserFleets={USER_FLEETS}
          selectedFleetId={1}
          onChange={noop}
        />,
        {
          contextValue: {
            isGlobalAdmin: true,
            config: {
              gitops: {
                gitops_mode_enabled: true,
                repository_url: "https://github.com/fleetdm/fleet",
              },
            } as any,
          },
        }
      );

      await user.click(getTrigger(/Fleet 1/));
      expect(
        screen.queryByRole("button", { name: /add fleet/i })
      ).not.toBeInTheDocument();
    });

    it("hides the button for global admins when rendered as a form field", async () => {
      const user = userEvent.setup();
      renderWithAppContext(
        <FleetsDropdown
          currentUserFleets={USER_FLEETS}
          selectedFleetId={1}
          onChange={noop}
          asFormField
        />,
        { contextValue: { isGlobalAdmin: true } }
      );

      await user.click(getTrigger(/Fleet 1/));
      expect(
        screen.queryByRole("button", { name: /add fleet/i })
      ).not.toBeInTheDocument();
    });

    it("hides the button for global admins when Primo mode is enabled", async () => {
      const user = userEvent.setup();
      renderWithAppContext(
        <FleetsDropdown
          currentUserFleets={USER_FLEETS}
          selectedFleetId={1}
          onChange={noop}
        />,
        {
          contextValue: {
            isGlobalAdmin: true,
            config: {
              partnerships: { enable_primo: true },
            } as any,
          },
        }
      );

      await user.click(getTrigger(/Fleet 1/));
      expect(
        screen.queryByRole("button", { name: /add fleet/i })
      ).not.toBeInTheDocument();
    });
  });
});
