import { screen } from "@testing-library/react";
import React, { useState } from "react";

import { IMdmVppToken } from "interfaces/mdm";
import {
  APP_CONTEXT_ALL_TEAMS_SUMMARY,
  APP_CONTEXT_NO_TEAM_SUMMARY,
} from "interfaces/team";
import { renderWithSetup } from "test/test-utils";

import {
  IVppFleetsValue,
  teamIdsForApi,
  vppFleetsValueFromToken,
} from "./helpers";
import VppFleetsField from "./VppFleetsField";

const AVAILABLE_TEAMS = [
  APP_CONTEXT_ALL_TEAMS_SUMMARY,
  { id: 1, name: "Workstations" },
  { id: 2, name: "Mobile" },
  APP_CONTEXT_NO_TEAM_SUMMARY,
];

const StatefulField = ({
  initialValue,
  onChangeSpy,
}: {
  initialValue: IVppFleetsValue;
  onChangeSpy?: (value: IVppFleetsValue) => void;
}) => {
  const [value, setValue] = useState(initialValue);
  return (
    <VppFleetsField
      value={value}
      onChange={(v) => {
        setValue(v);
        onChangeSpy?.(v);
      }}
      availableTeams={AVAILABLE_TEAMS}
      tooltip="Fleets tooltip"
      tooltipPosition="top"
      helpText="Unchecking a fleet deletes its apps."
    />
  );
};

describe("VppFleetsField", () => {
  it("hides the fleet list when All fleets is selected", () => {
    renderWithSetup(
      <StatefulField initialValue={{ mode: "all", teamIds: [] }} />
    );

    expect(screen.getByRole("radio", { name: "All fleets" })).toBeChecked();
    expect(screen.queryByPlaceholderText("Search fleets")).toBeNull();
    expect(screen.queryByText("Workstations")).toBeNull();
  });

  it("lists every fleet, including Unassigned but not All fleets, when Custom is selected", async () => {
    const { user } = renderWithSetup(
      <StatefulField initialValue={{ mode: "all", teamIds: [] }} />
    );

    await user.click(screen.getByRole("radio", { name: "Custom" }));

    expect(screen.getByPlaceholderText("Search fleets")).toBeInTheDocument();
    expect(
      screen.getByRole("checkbox", { name: "Workstations" })
    ).toBeVisible();
    expect(screen.getByRole("checkbox", { name: "Mobile" })).toBeVisible();
    expect(screen.getByRole("checkbox", { name: "Unassigned" })).toBeVisible();
    expect(screen.queryByRole("checkbox", { name: "All fleets" })).toBeNull();
    expect(
      screen.getByText("Unchecking a fleet deletes its apps.")
    ).toBeVisible();
  });

  it("adds a checked fleet as a pill and removes it when the pill is dismissed", async () => {
    const onChangeSpy = jest.fn();
    const { user } = renderWithSetup(
      <StatefulField
        initialValue={{ mode: "custom", teamIds: [] }}
        onChangeSpy={onChangeSpy}
      />
    );

    await user.click(screen.getByRole("checkbox", { name: "Mobile" }));
    expect(onChangeSpy).toHaveBeenLastCalledWith({
      mode: "custom",
      teamIds: [2],
    });

    await user.click(screen.getByRole("button", { name: "Remove Mobile" }));
    expect(onChangeSpy).toHaveBeenLastCalledWith({
      mode: "custom",
      teamIds: [],
    });
    expect(screen.getByRole("checkbox", { name: "Mobile" })).not.toBeChecked();
  });

  it("keeps fleets selected on other tokens selectable", () => {
    renderWithSetup(
      <StatefulField initialValue={{ mode: "custom", teamIds: [1] }} />
    );

    expect(
      screen.getByRole("checkbox", { name: "Workstations" })
    ).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Mobile" })).toHaveAttribute(
      "aria-disabled",
      "false"
    );
  });
});

describe("VppFleetsField helpers", () => {
  const token: IMdmVppToken = {
    id: 1,
    org_name: "Acme",
    location: "Acme",
    country_code: "us",
    renew_date: "2027-04-27T00:00:00Z",
    teams: null,
  };

  it("maps token assignments to the field value", () => {
    expect(vppFleetsValueFromToken({ ...token, teams: [] })).toEqual({
      mode: "all",
      teamIds: [],
    });
    expect(vppFleetsValueFromToken({ ...token, teams: null })).toEqual({
      mode: "custom",
      teamIds: [],
    });
    expect(
      vppFleetsValueFromToken({
        ...token,
        teams: [
          { team_id: 0, name: "Unassigned" },
          { team_id: 2, name: "Mobile" },
        ],
      })
    ).toEqual({ mode: "custom", teamIds: [0, 2] });
  });

  it("maps the field value to the API fleets value", () => {
    expect(teamIdsForApi({ mode: "all", teamIds: [1] })).toEqual([]);
    expect(teamIdsForApi({ mode: "custom", teamIds: [] })).toBeNull();
    expect(teamIdsForApi({ mode: "custom", teamIds: [0, 2] })).toEqual([0, 2]);
  });
});
