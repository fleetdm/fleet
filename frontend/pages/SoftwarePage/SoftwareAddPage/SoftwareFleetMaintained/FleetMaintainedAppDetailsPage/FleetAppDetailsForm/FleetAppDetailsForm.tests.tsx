import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";

import FleetAppDetailsForm from "./FleetAppDetailsForm";

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
    context: {
      app: {
        config: { gitops: { gitops_mode_enabled: gitOpsModeEnabled } },
      },
    },
  });
  return render(<FleetAppDetailsForm {...defaultProps} />);
};

describe("FleetAppDetailsForm", () => {
  beforeEach(() => jest.clearAllMocks());

  it("submits Self-service and Deploy selections", async () => {
    const { user } = renderForm();
    const selfServiceSwitch = screen
      .getByText("Self-service")
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
    expect(screen.getByRole("button", { name: "Add software" })).toBeDisabled();
  });
});
