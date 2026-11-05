import { screen, waitFor } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import mdmAPI from "services/entities/mdm";
import { createCustomRenderer } from "test/test-utils";

import BootstrapAdvancedOptions, {
  IBootstrapAdvancedOptionsFormData,
} from "./BootstrapAdvancedOptions";

const MANUAL_ENROLLMENT_LABEL = "Install during manual enrollment";
const MIGRATION_LABEL = "Install during migration";
const CUSTOM_AGENT_LABEL = "Deploy custom agent";

const allOff: IBootstrapAdvancedOptionsFormData = {
  macos_enable_bootstrap_package_during_manual_enrollment: false,
  macos_enable_bootstrap_package_during_migration: false,
  macos_manual_agent_install: false,
};

const defaultProps = {
  currentTeamId: 1,
  initialValues: allOff,
  hasUploadedPackage: true,
  hasSetupExperienceSoftwareOrScript: false,
  enrollSecret: "secret",
};

const render = createCustomRenderer({
  withBackendMock: true,
  context: {
    app: {
      config: createMockConfig({
        server_settings: {
          ...createMockConfig().server_settings,
          server_url: "https://fleet.example.com",
        },
      }),
    },
  },
});

describe("BootstrapAdvancedOptions", () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("checks the options from the initial values", async () => {
    const { user } = render(
      <BootstrapAdvancedOptions
        {...defaultProps}
        initialValues={{
          macos_enable_bootstrap_package_during_manual_enrollment: true,
          macos_enable_bootstrap_package_during_migration: false,
          macos_manual_agent_install: true,
        }}
      />
    );

    expect(screen.getByLabelText(MANUAL_ENROLLMENT_LABEL)).toBeChecked();
    expect(screen.getByLabelText(MIGRATION_LABEL)).not.toBeChecked();

    // the custom agent option is hidden until advanced options are revealed
    expect(screen.queryByLabelText(CUSTOM_AGENT_LABEL)).not.toBeInTheDocument();
    await user.click(screen.getByText("Advanced options"));
    expect(screen.getByLabelText(CUSTOM_AGENT_LABEL)).toBeChecked();
  });

  it("disables all options and saving when no bootstrap package is uploaded", async () => {
    const { user } = render(
      <BootstrapAdvancedOptions {...defaultProps} hasUploadedPackage={false} />
    );

    await user.click(screen.getByText("Advanced options"));

    expect(screen.getByLabelText(MANUAL_ENROLLMENT_LABEL)).toBeDisabled();
    expect(screen.getByLabelText(MIGRATION_LABEL)).toBeDisabled();
    expect(screen.getByLabelText(CUSTOM_AGENT_LABEL)).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("enables all options and saving when a bootstrap package is uploaded", async () => {
    const { user } = render(<BootstrapAdvancedOptions {...defaultProps} />);

    await user.click(screen.getByText("Advanced options"));

    expect(screen.getByLabelText(MANUAL_ENROLLMENT_LABEL)).toBeEnabled();
    expect(screen.getByLabelText(MIGRATION_LABEL)).toBeEnabled();
    expect(screen.getByLabelText(CUSTOM_AGENT_LABEL)).toBeEnabled();
    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("only disables the custom agent option when there is setup experience software or a script", async () => {
    const { user } = render(
      <BootstrapAdvancedOptions
        {...defaultProps}
        hasSetupExperienceSoftwareOrScript
      />
    );

    await user.click(screen.getByText("Advanced options"));

    expect(screen.getByLabelText(CUSTOM_AGENT_LABEL)).toBeDisabled();
    expect(screen.getByLabelText(MANUAL_ENROLLMENT_LABEL)).toBeEnabled();
    expect(screen.getByLabelText(MIGRATION_LABEL)).toBeEnabled();
    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("saves all options for the fleet", async () => {
    const updateSpy = jest
      .spyOn(mdmAPI, "updateSetupExperienceSettings")
      .mockResolvedValue({});
    const { user } = render(<BootstrapAdvancedOptions {...defaultProps} />);

    await user.click(screen.getByText(MANUAL_ENROLLMENT_LABEL));
    await user.click(screen.getByText(MIGRATION_LABEL));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(updateSpy).toHaveBeenCalledWith({
        fleet_id: 1,
        macos_enable_bootstrap_package_during_manual_enrollment: true,
        macos_enable_bootstrap_package_during_migration: true,
        macos_manual_agent_install: false,
      });
    });
  });

  it("resets the form when the initial values change", async () => {
    const { user, rerender } = render(
      <BootstrapAdvancedOptions {...defaultProps} />
    );

    await user.click(screen.getByText(MIGRATION_LABEL));
    expect(screen.getByLabelText(MIGRATION_LABEL)).toBeChecked();

    // e.g. after the bootstrap package is deleted and the config is refetched
    rerender(
      <BootstrapAdvancedOptions
        {...defaultProps}
        initialValues={{ ...allOff }}
      />
    );

    expect(screen.getByLabelText(MIGRATION_LABEL)).not.toBeChecked();
  });

  it("links to the enrollment link in the manual enrollment tooltip when there is an enroll secret", async () => {
    const { user } = render(
      <BootstrapAdvancedOptions {...defaultProps} enrollSecret="a/b+c" />
    );

    await user.hover(screen.getByText(MANUAL_ENROLLMENT_LABEL));

    const link = await screen.findByRole("link", { name: /enrollment link/ });
    expect(link).toHaveAttribute(
      "href",
      "https://fleet.example.com/enroll?enroll_secret=a%2Fb%2Bc"
    );
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("links to adding an enroll secret in the manual enrollment tooltip when there is none", async () => {
    const { user } = render(
      <BootstrapAdvancedOptions {...defaultProps} enrollSecret={null} />
    );

    await user.hover(screen.getByText(MANUAL_ENROLLMENT_LABEL));

    const link = await screen.findByRole("link", {
      name: /add an enroll secret/,
    });
    expect(link).toHaveAttribute(
      "href",
      "/hosts/manage?manage_enroll_secrets=1&fleet_id=1"
    );
    expect(
      screen.queryByRole("link", { name: /enrollment link/ })
    ).not.toBeInTheDocument();
  });
});
