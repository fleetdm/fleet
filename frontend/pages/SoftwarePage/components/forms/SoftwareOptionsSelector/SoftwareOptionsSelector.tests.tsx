import React from "react";
import { screen, fireEvent } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import SoftwareOptionsSelector from "./SoftwareOptionsSelector";

const defaultProps = {
  formData: {
    selfService: false,
    automaticInstall: false,
    targetType: "",
    customTarget: "",
    labelTargets: {},
    selectedApp: null,
    categories: [],
  },
  onToggleAutomaticInstall: jest.fn(),
  onToggleSelfService: jest.fn(),
  onSelectCategory: jest.fn(),
  onClickPreviewEndUserExperience: jest.fn(),
};

describe("SoftwareOptionsSelector", () => {
  const renderComponent = (props = {}) => {
    return createCustomRenderer({ context: {} })(
      <SoftwareOptionsSelector {...defaultProps} {...props} />
    );
  };

  it("calls onToggleSelfService when the self-service checkbox is toggled", () => {
    const onToggleSelfService = jest.fn();
    renderComponent({ onToggleSelfService });

    const selfServiceCheckbox = screen
      .getByText("Self-service")
      .closest('div[role="checkbox"]');
    if (selfServiceCheckbox) {
      fireEvent.click(selfServiceCheckbox);
    } else {
      throw new Error("Self-service checkbox not found");
    }

    expect(onToggleSelfService).toHaveBeenCalledTimes(1);
    expect(onToggleSelfService).toHaveBeenCalledWith(true);
  });

  it("calls onToggleAutomaticInstall when the automatic install checkbox is toggled", () => {
    const onToggleAutomaticInstall = jest.fn();
    renderComponent({ onToggleAutomaticInstall });

    const automaticInstallCheckbox = screen
      .getByText("Automatic install")
      .closest('div[role="checkbox"]');
    if (automaticInstallCheckbox) {
      fireEvent.click(automaticInstallCheckbox);
    } else {
      throw new Error("Automatic install checkbox not found");
    }

    expect(onToggleAutomaticInstall).toHaveBeenCalledTimes(1);
    expect(onToggleAutomaticInstall).toHaveBeenCalledWith(true);
  });

  it("disables self-service and automatic install checkboxes for iOS", () => {
    renderComponent({ platform: "ios" });

    // Targeting the checkbox elements directly
    const selfServiceCheckbox = screen
      .getByText("Self-service")
      .closest('[role="checkbox"]');
    const automaticInstallCheckbox = screen
      .getByText("Automatic install")
      .closest('[role="checkbox"]');

    expect(selfServiceCheckbox).toHaveAttribute("aria-disabled", "true");
    expect(automaticInstallCheckbox).toHaveAttribute("aria-disabled", "true");
  });

  it("disables self-service and automatic install checkboxes for iPadOS", () => {
    renderComponent({ platform: "ipados" });

    // Targeting the checkbox elements directly
    const selfServiceCheckbox = screen
      .getByText("Self-service")
      .closest('[role="checkbox"]');
    const automaticInstallCheckbox = screen
      .getByText("Automatic install")
      .closest('[role="checkbox"]');

    expect(selfServiceCheckbox).toHaveAttribute("aria-disabled", "true");
    expect(automaticInstallCheckbox).toHaveAttribute("aria-disabled", "true");
  });

  it("disables checkboxes when disableOptions is true", () => {
    renderComponent({ disableOptions: true });

    const selfServiceCheckbox = screen
      .getByText("Self-service")
      .closest('[role="checkbox"]');
    const automaticInstallCheckbox = screen
      .getByText("Automatic install")
      .closest('[role="checkbox"]');

    expect(selfServiceCheckbox).toHaveAttribute("aria-disabled", "true");
    expect(automaticInstallCheckbox).toHaveAttribute("aria-disabled", "true");
  });

  it("renders the InfoBanner when automaticInstall is true and isCustomPackage is true", () => {
    renderComponent({
      formData: { ...defaultProps.formData, automaticInstall: true },
      isCustomPackage: true,
    });

    expect(
      screen.getByText(
        /Installing software over existing installations might cause issues/i
      )
    ).toBeInTheDocument();
  });

  it("does not render the InfoBanner when automaticInstall is false", () => {
    renderComponent({
      formData: { ...defaultProps.formData, automaticInstall: false },
      isCustomPackage: true,
    });

    expect(
      screen.queryByText(
        /Installing software over existing installations might cause issues/i
      )
    ).not.toBeInTheDocument();
  });

  it("does not render the InfoBanner when isCustomPackage is false", () => {
    renderComponent({
      formData: { ...defaultProps.formData, automaticInstall: true },
      isCustomPackage: false,
    });

    expect(
      screen.queryByText(
        /Installing software over existing installations might cause issues/i
      )
    ).not.toBeInTheDocument();
  });

  it("does not render automatic install checkbox when isEditingSoftware is true", () => {
    renderComponent({ isEditingSoftware: true });

    expect(screen.queryByText("Automatic install")).not.toBeInTheDocument();
  });

  it("displays platform-specific message for iOS", () => {
    renderComponent({ platform: "ios" });

    expect(
      screen.getByText(
        /Currently, self-service and automatic installation are not available for iOS and iPadOS/i
      )
    ).toBeInTheDocument();
  });

  it("displays platform-specific message for iPadOS", () => {
    renderComponent({ platform: "ipados" });

    expect(
      screen.getByText(
        /Currently, self-service and automatic installation are not available for iOS and iPadOS/i
      )
    ).toBeInTheDocument();
  });

  it("does not render automatic install checkbox in edit mode", () => {
    renderComponent({ isEditingSoftware: true });

    expect(screen.queryByText("Automatic install")).not.toBeInTheDocument();
  });
});
