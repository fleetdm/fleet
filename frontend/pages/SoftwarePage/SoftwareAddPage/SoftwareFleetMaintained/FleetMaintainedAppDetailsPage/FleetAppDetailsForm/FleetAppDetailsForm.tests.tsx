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

const renderForm = (gitOpsModeEnabled = false, platform?: string) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        config: { gitops: { gitops_mode_enabled: gitOpsModeEnabled } },
      },
    },
  });
  return render(<FleetAppDetailsForm {...defaultProps} platform={platform} />);
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

  it("submits endUserExperience: notify when the dropdown is set on macOS", async () => {
    const { user } = renderForm(false, "darwin");

    await user.click(screen.getByRole("checkbox", { name: "patch" }));
    await user.click(screen.getByRole("radio", { name: "Force patch" }));
    // react-select-5 opens on mouseDown of the wrapper; findByText waits for
    // the option to portal in.
    await user.click(screen.getByRole("combobox"));
    const notifyOption = await screen.findByText("Notify before patching");
    await user.click(notifyOption);
    await user.click(screen.getByRole("button", { name: "Add software" }));

    expect(defaultProps.onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        patch: true,
        patchOption: "force",
        endUserExperience: "notify",
      })
    );
  });

  it("does not render the End user experience dropdown for a Windows FMA", async () => {
    const { user } = renderForm(false, "windows");

    await user.click(screen.getByRole("checkbox", { name: "patch" }));
    await user.click(screen.getByRole("radio", { name: "Force patch" }));

    expect(
      screen.queryByRole("combobox", { name: /End user experience/i })
    ).not.toBeInTheDocument();
    // Windows-specific coming-soon banner replaces the dropdown + link.
    expect(
      screen.getByText(/Notifications for Windows are coming soon\./)
    ).toBeInTheDocument();
  });

  it("locks the pre-install query as Fleet-managed when Force + Notify is selected on macOS", async () => {
    const { user } = renderForm(false, "darwin");

    await user.click(screen.getByRole("checkbox", { name: "patch" }));
    await user.click(screen.getByRole("radio", { name: "Force patch" }));
    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByText("Notify before patching"));

    await user.click(screen.getByRole("button", { name: /advanced options/i }));
    expect(
      await screen.findByText(
        /Pre-install query won't run when install is triggered via self-service/
      )
    ).toBeInTheDocument();
  });

  it("keeps the pre-install query editable when Force patch is selected on a Windows FMA", async () => {
    // Defense-in-depth: the Add FMA form starts with endUserExperience
    // "immediate" and Windows hides the dropdown, so state can't reach
    // "notify" through the UI. The platform gate on the `patchWhenClosed`
    // composite is what still keeps the pre-install query editable — this
    // test locks in that Windows Force-patch stays unlocked.
    const { user } = renderForm(false, "windows");

    await user.click(screen.getByRole("checkbox", { name: "patch" }));
    await user.click(screen.getByRole("radio", { name: "Force patch" }));

    await user.click(screen.getByRole("button", { name: /advanced options/i }));
    expect(
      screen.queryByText(
        /Pre-install query won't run when install is triggered via self-service/
      )
    ).not.toBeInTheDocument();
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
