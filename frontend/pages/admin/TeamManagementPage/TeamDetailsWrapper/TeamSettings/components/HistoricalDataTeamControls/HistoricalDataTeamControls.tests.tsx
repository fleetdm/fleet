import React from "react";
import { screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import HistoricalDataTeamControls from "./HistoricalDataTeamControls";

const renderControls = (
  overrides: Partial<
    React.ComponentProps<typeof HistoricalDataTeamControls>
  > = {}
) => {
  const onChange = jest.fn();
  const utils = renderWithSetup(
    <HistoricalDataTeamControls
      disableHostsActive={false}
      disableVulnerabilities={false}
      globalHostsActiveDisabled={false}
      globalVulnerabilitiesDisabled={false}
      onChange={onChange}
      {...overrides}
    />
  );
  return { ...utils, onChange };
};

describe("HistoricalDataTeamControls", () => {
  it("renders the section heading and both checkboxes", () => {
    renderControls();
    expect(screen.getByText("Activity & data retention")).toBeInTheDocument();
    expect(screen.getByLabelText(/Disable hosts online/i)).toBeInTheDocument();
    expect(
      screen.getByLabelText(/Disable vulnerabilities/i)
    ).toBeInTheDocument();
  });

  it("reflects the team's stored disable values", () => {
    renderControls({
      disableHostsActive: true,
      disableVulnerabilities: false,
    });
    expect(screen.getByLabelText(/Disable hosts online/i)).toBeChecked();
    expect(screen.getByLabelText(/Disable vulnerabilities/i)).not.toBeChecked();
  });

  it("locks the hosts-active checkbox when global is disabled", () => {
    renderControls({ globalHostsActiveDisabled: true });
    expect(screen.getByLabelText(/Disable hosts online/i)).toBeDisabled();
    // The vulnerabilities checkbox stays interactive
    expect(
      screen.getByLabelText(/Disable vulnerabilities/i)
    ).not.toBeDisabled();
  });

  it("locks the vulnerabilities checkbox when global is disabled", () => {
    renderControls({ globalVulnerabilitiesDisabled: true });
    expect(screen.getByLabelText(/Disable vulnerabilities/i)).toBeDisabled();
    expect(screen.getByLabelText(/Disable hosts online/i)).not.toBeDisabled();
  });

  it("preserves the team's stored value while locked", () => {
    renderControls({
      disableHostsActive: true,
      globalHostsActiveDisabled: true,
    });
    const checkbox = screen.getByLabelText(/Disable hosts online/i);
    expect(checkbox).toBeChecked();
    expect(checkbox).toBeDisabled();
  });

  it("calls onChange when an enabled checkbox is toggled", async () => {
    const { user, onChange } = renderControls();
    await user.click(
      screen.getByRole("checkbox", { name: "disableHostsActive" })
    );
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "disableHostsActive",
        value: true,
      })
    );
  });

  it("does not call onChange when a locked checkbox is clicked", async () => {
    const { user, onChange } = renderControls({
      globalHostsActiveDisabled: true,
    });
    await user.click(
      screen.getByRole("checkbox", { name: "disableHostsActive" })
    );
    expect(onChange).not.toHaveBeenCalled();
  });
});
