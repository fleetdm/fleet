import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import labelsAPI from "services/entities/labels";
import { ILabelSummary } from "interfaces/label";

import FleetAppDetailsForm from "./FleetAppDetailsForm";

const mockLabels: ILabelSummary[] = [
  { id: 1, name: "Engineering", label_type: "regular" },
  { id: 2, name: "Sales", label_type: "regular" },
];

const defaultProps: React.ComponentProps<typeof FleetAppDetailsForm> = {
  categories: [],
  defaultInstallScript: "install",
  defaultPostInstallScript: "post-install",
  defaultUninstallScript: "uninstall",
  teamId: "1",
  onCancel: jest.fn(),
  onSubmit: jest.fn(),
};

const renderForm = (gitOpsModeEnabled = false) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        config: { gitops: { gitops_mode_enabled: gitOpsModeEnabled } },
      },
    },
  });
  return render(<FleetAppDetailsForm {...defaultProps} />);
};

describe("FleetAppDetailsForm", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(labelsAPI, "summary").mockResolvedValue({ labels: mockLabels });
  });

  it("submits Self-service and Deploy selections", async () => {
    const { user } = renderForm();
    const selfServiceSwitch = screen
      .getByText("Self service")
      .closest(".fleet-slider__wrapper")
      ?.querySelector('[role="switch"]');
    expect(selfServiceSwitch).not.toBeNull();

    await user.click(selfServiceSwitch as Element);
    await user.click(screen.getByRole("checkbox", { name: "force-install" }));
    await user.click(screen.getByRole("checkbox", { name: "patch" }));
    await user.click(screen.getByRole("radio", { name: "Force patch" }));
    await user.click(screen.getByRole("button", { name: "Add software" }));

    expect(defaultProps.onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        selfService: true,
        forceInstall: true,
        patch: true,
        patchOption: "force",
      })
    );
  });

  it("disables Deploy and Add software in GitOps mode", () => {
    renderForm(true);

    expect(
      screen.getByRole("checkbox", { name: "force-install" })
    ).toHaveAttribute("aria-disabled", "true");
    expect(screen.getByRole("checkbox", { name: "patch" })).toHaveAttribute(
      "aria-disabled",
      "true"
    );
    expect(screen.getByRole("radio", { name: "All hosts" })).toBeDisabled();
    expect(screen.getByRole("radio", { name: "Custom" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Add software" })).toBeDisabled();
  });

  it("targets Custom labels and gates submit on a label selection", async () => {
    const { user } = renderForm();

    // Target section renders with All hosts selected by default.
    expect(screen.getByText("Target")).toBeInTheDocument();
    const addButton = screen.getByRole("button", { name: "Add software" });
    expect(addButton).toBeEnabled();

    // Custom with no label selected is invalid → submit disabled.
    await user.click(screen.getByRole("radio", { name: "Custom" }));
    expect(addButton).toBeDisabled();

    // Selecting a label (loaded from the labels summary) re-enables submit.
    const engineering = await screen.findByRole("checkbox", {
      name: "Engineering",
    });
    await user.click(engineering);
    expect(addButton).toBeEnabled();

    await user.click(addButton);
    expect(defaultProps.onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        targetType: "Custom",
        labelTargets: expect.objectContaining({ Engineering: true }),
      })
    );
  });

  it("reveals Advanced options with a pre-install query field", async () => {
    const { user } = renderForm();

    const revealButton = screen.getByRole("button", {
      name: /advanced options/i,
    });
    expect(revealButton).toBeInTheDocument();

    await user.click(revealButton);
    expect(await screen.findByText("Pre-install query")).toBeInTheDocument();
  });
});
