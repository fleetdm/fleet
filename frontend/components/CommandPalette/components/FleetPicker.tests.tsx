import React from "react";
import { screen } from "@testing-library/react";
import { Command } from "cmdk";
import { createCustomRenderer } from "test/test-utils";

import FleetPicker from "./FleetPicker";

// cmdk uses scrollIntoView which JSDOM doesn't implement
Element.prototype.scrollIntoView = jest.fn();

const renderInCommand = createCustomRenderer();
const renderPicker = (
  component: React.ReactElement
): ReturnType<typeof renderInCommand> =>
  renderInCommand(<Command>{component}</Command>);

describe("FleetPicker", () => {
  const availableTeams = [
    { id: -1, name: "All fleets" },
    { id: 0, name: "No team" },
    { id: 1, name: "Engineering" },
    { id: 2, name: "Sales" },
  ];

  it("renders every fleet in availableTeams", () => {
    renderPicker(
      <FleetPicker
        availableTeams={availableTeams}
        currentTeam={availableTeams[2]}
        onSelect={jest.fn()}
      />
    );

    expect(screen.getByText("All fleets")).toBeInTheDocument();
    expect(screen.getByText("No team")).toBeInTheDocument();
    expect(screen.getByText("Engineering")).toBeInTheDocument();
    expect(screen.getByText("Sales")).toBeInTheDocument();
  });

  it("marks the current fleet with the selected modifier class", () => {
    renderPicker(
      <FleetPicker
        availableTeams={availableTeams}
        currentTeam={availableTeams[2]}
        onSelect={jest.fn()}
      />
    );

    const engineering = screen.getByText("Engineering");
    expect(engineering.className).toMatch(/__item-label--selected/);

    const sales = screen.getByText("Sales");
    expect(sales.className).not.toMatch(/__item-label--selected/);
  });

  it("calls onSelect with the fleet id when an item is clicked", async () => {
    const onSelect = jest.fn();
    const { user } = renderPicker(
      <FleetPicker
        availableTeams={availableTeams}
        currentTeam={availableTeams[2]}
        onSelect={onSelect}
      />
    );

    await user.click(screen.getByText("Sales"));
    expect(onSelect).toHaveBeenCalledWith(2);
  });

  it("renders empty (no items) when availableTeams is undefined", () => {
    renderPicker(<FleetPicker onSelect={jest.fn()} />);
    expect(screen.queryByText("Engineering")).not.toBeInTheDocument();
  });
});
