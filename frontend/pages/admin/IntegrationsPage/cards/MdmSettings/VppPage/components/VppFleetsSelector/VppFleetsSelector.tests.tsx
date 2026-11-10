import { screen, waitFor } from "@testing-library/react";
import React, { useState } from "react";

import { createMockTeamSummary } from "__mocks__/teamMock";
import { createCustomRenderer } from "test/test-utils";

import { IVppFleetsValue } from "./helpers";
import VppFleetsSelector from "./VppFleetsSelector";

const HELP_TEXT = /If you uncheck a fleet, apps from this VPP token will be/;

const availableTeams = [
  createMockTeamSummary({ id: -1, name: "All fleets" }),
  createMockTeamSummary({ id: 1, name: "Workstations" }),
  createMockTeamSummary({ id: 2, name: "Personal mobile devices" }),
  createMockTeamSummary({ id: 0, name: "Unassigned" }),
];

const render = createCustomRenderer({
  context: { app: { availableTeams } },
});

const Harness = ({
  initial,
  showUncheckHelpText,
  onChangeSpy,
}: {
  initial: IVppFleetsValue;
  showUncheckHelpText?: boolean;
  onChangeSpy?: (value: IVppFleetsValue) => void;
}) => {
  const [value, setValue] = useState(initial);
  return (
    <VppFleetsSelector
      value={value}
      onChange={(v) => {
        onChangeSpy?.(v);
        setValue(v);
      }}
      tooltipPosition="top-start"
      showUncheckHelpText={showUncheckHelpText}
    />
  );
};

describe("VppFleetsSelector", () => {
  it("shows only the radios under All fleets", () => {
    render(<Harness initial={{ allFleets: true, fleetIds: [] }} />);

    expect(screen.getByRole("radio", { name: "All fleets" })).toBeChecked();
    expect(screen.getByRole("radio", { name: "Custom" })).not.toBeChecked();
    expect(screen.queryByPlaceholderText("Search fleets")).toBeNull();
    expect(screen.queryByRole("checkbox")).toBeNull();
  });

  it("lists every fleet plus Unassigned under Custom, and not the All fleets entry", () => {
    render(<Harness initial={{ allFleets: false, fleetIds: [] }} />);

    expect(screen.getByRole("radio", { name: "Custom" })).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: "Workstations" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("checkbox", { name: "Personal mobile devices" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("checkbox", { name: "Unassigned" })
    ).toBeInTheDocument();
    expect(screen.getAllByRole("checkbox")).toHaveLength(3);
  });

  it("switching to Custom and back to All fleets keeps the checks", async () => {
    const { user } = render(
      <Harness initial={{ allFleets: false, fleetIds: [1] }} />
    );

    await user.click(screen.getByRole("radio", { name: "All fleets" }));
    expect(screen.queryByRole("checkbox")).toBeNull();

    await user.click(screen.getByRole("radio", { name: "Custom" }));
    expect(
      screen.getByRole("checkbox", { name: "Workstations" })
    ).toBeChecked();
  });

  it("filters the checklist by name, ignoring case", async () => {
    const { user } = render(
      <Harness initial={{ allFleets: false, fleetIds: [] }} />
    );

    await user.type(screen.getByPlaceholderText("Search fleets"), "MOB");

    await waitFor(() => {
      expect(
        screen.queryByRole("checkbox", { name: "Workstations" })
      ).toBeNull();
    });
    expect(
      screen.getByRole("checkbox", { name: "Personal mobile devices" })
    ).toBeInTheDocument();
  });

  it("checking a fleet adds a pill and reports the new ids", async () => {
    const onChangeSpy = jest.fn();
    const { user } = render(
      <Harness
        initial={{ allFleets: false, fleetIds: [] }}
        onChangeSpy={onChangeSpy}
      />
    );

    await user.click(screen.getByRole("checkbox", { name: "Workstations" }));

    expect(onChangeSpy).toHaveBeenLastCalledWith({
      allFleets: false,
      fleetIds: [1],
    });
    expect(
      screen.getByRole("button", { name: "Remove Workstations" })
    ).toBeInTheDocument();
  });

  it("removing a pill unchecks its fleet", async () => {
    const { user } = render(
      <Harness initial={{ allFleets: false, fleetIds: [1, 2] }} />
    );

    await user.click(
      screen.getByRole("button", { name: "Remove Workstations" })
    );

    expect(
      screen.queryByRole("button", { name: "Remove Workstations" })
    ).toBeNull();
    expect(
      screen.getByRole("checkbox", { name: "Workstations" })
    ).not.toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: "Personal mobile devices" })
    ).toBeChecked();
  });

  it("shows the uncheck help text under Custom, not under All fleets", async () => {
    const { user } = render(
      <Harness
        initial={{ allFleets: false, fleetIds: [] }}
        showUncheckHelpText
      />
    );
    expect(screen.getByText(HELP_TEXT)).toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: "All fleets" }));
    expect(screen.queryByText(HELP_TEXT)).toBeNull();
  });

  it("hides the uncheck help text without showUncheckHelpText", () => {
    render(<Harness initial={{ allFleets: false, fleetIds: [] }} />);
    expect(screen.queryByText(HELP_TEXT)).toBeNull();
  });

  it("keeps the search text and the filter when toggling All fleets and back to Custom", async () => {
    const { user } = render(
      <Harness initial={{ allFleets: false, fleetIds: [] }} />
    );

    await user.type(screen.getByPlaceholderText("Search fleets"), "MOB");
    await waitFor(() => {
      expect(
        screen.queryByRole("checkbox", { name: "Workstations" })
      ).toBeNull();
    });

    await user.click(screen.getByRole("radio", { name: "All fleets" }));
    await user.click(screen.getByRole("radio", { name: "Custom" }));

    expect(screen.getByPlaceholderText("Search fleets")).toHaveValue("MOB");
    expect(screen.queryByRole("checkbox", { name: "Workstations" })).toBeNull();
    expect(
      screen.getByRole("checkbox", { name: "Personal mobile devices" })
    ).toBeInTheDocument();
  });

  it("shows the Fleets tooltip copy on hover", async () => {
    const { user } = render(
      <Harness initial={{ allFleets: true, fleetIds: [] }} />
    );

    await user.hover(screen.getByText("Fleets"));

    const tooltip = await screen.findByRole("tooltip");
    expect(tooltip.textContent).toBe(
      "Choose Custom if you want your App Store apps from this VPP token to only be available to install on hosts in specific fleets."
    );
  });
});
