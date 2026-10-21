import { screen } from "@testing-library/react";
import React from "react";

import { renderWithSetup } from "test/test-utils";

import ProfileBottomBar from "./ProfileBottomBar";

describe("ProfileBottomBar", () => {
  it("says the profile applies to all hosts in the fleet", () => {
    renderWithSetup(
      <ProfileBottomBar
        targetType="All hosts"
        teamName="Workstations"
        submitButton={<button type="submit">Add profile</button>}
      />
    );

    expect(screen.getByText(/Applies to/)).toHaveTextContent(
      "Applies to all hosts for Workstations."
    );
    expect(
      screen.getByRole("button", { name: "Add profile" })
    ).toBeInTheDocument();
  });

  it("says custom targets and leaves out a missing fleet", () => {
    renderWithSetup(
      <ProfileBottomBar targetType="Custom" submitButton={null} />
    );

    expect(screen.getByText(/Applies to/)).toHaveTextContent(
      "Applies to hosts in custom targets."
    );
  });

  it("offers Edit targets only when it can be handled", async () => {
    const onEditTargets = jest.fn();
    const { user, rerender } = renderWithSetup(
      <ProfileBottomBar targetType="All hosts" submitButton={null} />
    );
    expect(
      screen.queryByRole("button", { name: "Edit targets" })
    ).not.toBeInTheDocument();

    rerender(
      <ProfileBottomBar
        targetType="All hosts"
        onEditTargets={onEditTargets}
        submitButton={null}
      />
    );
    await user.click(screen.getByRole("button", { name: "Edit targets" }));
    expect(onEditTargets).toHaveBeenCalledTimes(1);
  });
});
