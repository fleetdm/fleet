import React from "react";
import { render, screen } from "@testing-library/react";

import SoftwareDeploySelector, { PatchOption } from "./SoftwareDeploySelector";

const renderSelector = (
  overrides: Partial<React.ComponentProps<typeof SoftwareDeploySelector>> = {}
) => {
  const props: React.ComponentProps<typeof SoftwareDeploySelector> = {
    forceInstall: false,
    patch: false,
    patchOption: "closed",
    onToggleForceInstall: jest.fn(),
    onTogglePatch: jest.fn(),
    onSelectPatchOption: jest.fn(),
    ...overrides,
  };
  return { ...render(<SoftwareDeploySelector {...props} />), props };
};

describe("SoftwareDeploySelector", () => {
  it("shows Force install without patch options", () => {
    renderSelector({ forceInstall: true });

    expect(
      screen.getByRole("checkbox", { name: "force-install" })
    ).toBeChecked();
    expect(screen.queryByRole("radio")).not.toBeInTheDocument();
  });

  it("shows patch options with Patch when app is closed selected by default", () => {
    renderSelector({ patch: true });

    expect(screen.getByRole("checkbox", { name: "patch" })).toBeChecked();
    expect(
      screen.getByRole("radio", { name: "Patch when app is closed" })
    ).toBeChecked();
  });

  it("supports Force install and Patch together", () => {
    renderSelector({ forceInstall: true, patch: true });

    expect(
      screen.getByRole("checkbox", { name: "force-install" })
    ).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "patch" })).toBeChecked();
    expect(screen.getAllByRole("radio")).toHaveLength(3);
  });

  it("shows the Force patch information banner", () => {
    renderSelector({ patch: true, patchOption: "force" });

    expect(
      screen.getByText(
        "End user is not notified. Patch is forced as soon as policy fails. Notifications are coming soon."
      )
    ).toBeInTheDocument();
  });

  it("shows the pre-install query override notice in the Deploy modal", () => {
    renderSelector({ patch: true, showPatchWhenClosedNotice: true });

    expect(
      screen.getByText(/overrides the pre-install query \(advanced option\)/)
    ).toBeInTheDocument();
  });

  it("reports the selected patch option", async () => {
    const onSelectPatchOption = jest.fn<void, [PatchOption]>();
    const { getByRole } = renderSelector({
      patch: true,
      onSelectPatchOption,
    });

    getByRole("radio", { name: "Force patch" }).click();
    expect(onSelectPatchOption).toHaveBeenCalledWith("force");
  });
});
