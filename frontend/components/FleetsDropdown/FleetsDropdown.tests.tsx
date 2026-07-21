import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { noop } from "lodash";
// TODOL Replace renderWithAppContext with createCustomRenderer
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

  it("renders the given selected fleet from selectedTeamId", () => {
    render(
      <FleetsDropdown
        currentUserTeams={USER_FLEETS}
        selectedTeamId={1}
        onChange={noop}
      />
    );

    expect(getTrigger(/Fleet 1/)).toBeInTheDocument();
  });

  it("renders the first fleet option when includeAllTeams is false and when no selectedTeamId is given", () => {
    render(
      <FleetsDropdown
        currentUserTeams={USER_FLEETS}
        includeAllTeams={false}
        onChange={noop}
      />
    );

    expect(getTrigger(/Fleet 1/)).toBeInTheDocument();
  });

  describe("user is on the global team", () => {
    const contextValue = {
      isOnGlobalTeam: true,
    };

    it("renders 'All fleets' when no selectedTeamId is given", () => {
      renderWithAppContext(
        <FleetsDropdown currentUserTeams={USER_FLEETS} onChange={noop} />,
        { contextValue }
      );

      expect(getTrigger(/All fleets/)).toBeInTheDocument();
    });

    it("renders the first fleet option when includeAllTeams is false and when no selectedTeamId is given", () => {
      renderWithAppContext(
        <FleetsDropdown
          currentUserTeams={USER_FLEETS}
          includeAllTeams={false}
          onChange={noop}
        />,
        { contextValue }
      );

      expect(getTrigger(/Fleet 1/)).toBeInTheDocument();
    });
  });

  describe("user is not on the global team", () => {
    const contextValue = { isOnGlobalTeam: false };
    const filteredUserFleets = USER_FLEETS.filter(
      (t) => t.id > APP_CONTEXT_NO_TEAM_ID
    );

    it("renders the first fleet when no selectedTeamId is given", () => {
      renderWithAppContext(
        <FleetsDropdown
          currentUserTeams={filteredUserFleets}
          onChange={noop}
        />,
        { contextValue }
      );

      expect(getTrigger(/Fleet 1/)).toBeInTheDocument();
    });
  });

  describe("in-menu search", () => {
    // Search only appears once the list has 10+ fleets.
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

    it("hides the search input when there are fewer than 10 fleets", async () => {
      const user = userEvent.setup();
      render(
        <FleetsDropdown
          currentUserTeams={USER_FLEETS}
          selectedTeamId={1}
          onChange={noop}
        />
      );

      await user.click(getTrigger(/Fleet 1/));
      expect(
        screen.queryByPlaceholderText("Search fleets")
      ).not.toBeInTheDocument();
    });

    it("renders a search input when there are 10 or more fleets", async () => {
      const user = userEvent.setup();
      render(
        <FleetsDropdown
          currentUserTeams={MANY_FLEETS}
          selectedTeamId={1}
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
          currentUserTeams={MANY_FLEETS}
          selectedTeamId={1}
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
          currentUserTeams={MANY_FLEETS}
          selectedTeamId={1}
          onChange={noop}
        />
      );

      await user.click(getTrigger(/Fleet 1/));
      fireEvent.change(screen.getByPlaceholderText("Search fleets"), {
        target: { value: "nothing-matches-this" },
      });

      expect(screen.getByText("No matching fleets")).toBeInTheDocument();
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
          currentUserTeams={USER_FLEETS}
          selectedTeamId={1}
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
          currentUserTeams={USER_FLEETS}
          selectedTeamId={1}
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
          currentUserTeams={MANY_FLEETS}
          selectedTeamId={1}
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
  });
});
