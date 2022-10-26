import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";

import { renderWithSetup } from "test/test-utils";
import { userTeamStub } from "test/stubs";
import SelectedTeamsForm from "./SelectedTeamsForm";

describe("SelectedTeamsForm - component", () => {
  it("correctly renders checkboxes users current teams", () => {
    const currentTeam = userTeamStub;
    const teamNotOn = { ...userTeamStub, id: 2, name: "Not Selected Team" };
    render(
      <SelectedTeamsForm
        availableTeams={[currentTeam, teamNotOn]}
        usersCurrentTeams={[currentTeam]}
        onFormChange={noop}
      />
    );

    const checkbox = screen.getByRole("checkbox", { name: "Test Team" });
    const notSelectedCheckbox = screen.getByRole("checkbox", {
      name: "Not Selected Team",
    });
    expect(checkbox).toBeChecked();
    expect(notSelectedCheckbox).not.toBeChecked();
  });

  it("Correctly passes up selected teams to parent when one of the checkboxes is changed", async () => {
    const onChangeStub = jest.fn();
    const currentTeam = userTeamStub;
    const teamNotOn = { ...userTeamStub, id: 2, name: "Not Selected Team" };
    const { user } = renderWithSetup(
      <SelectedTeamsForm
        availableTeams={[currentTeam, teamNotOn]}
        usersCurrentTeams={[currentTeam]}
        onFormChange={onChangeStub}
      />
    );

    // checking an unselected team.
    const notSelectedCheckbox = screen.getByRole("checkbox", {
      name: "Not Selected Team",
    });
    await user.click(notSelectedCheckbox);
    expect(onChangeStub).toHaveBeenCalledWith([currentTeam, teamNotOn]);
    expect(notSelectedCheckbox).toBeChecked();

    // unchecking a selected team.
    const recentlySelectedCheckbox = screen.getByRole("checkbox", {
      name: "Not Selected Team",
    });
    await user.click(recentlySelectedCheckbox);
    expect(onChangeStub).toHaveBeenCalledWith([currentTeam]);
    expect(notSelectedCheckbox).not.toBeChecked();
  });
});
