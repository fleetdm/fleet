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

const getSwitchByLabelText = (text: string) => {
  const label = screen.getByText(text);
  const wrapper = label.closest(".fleet-slider__wrapper");
  if (!wrapper) throw new Error(`Wrapper not found for "${text}"`);
  const btn = wrapper.querySelector('button[role="switch"]');
  if (!btn) throw new Error(`Switch button not found for "${text}"`);
  return btn as HTMLButtonElement;
};

describe("SoftwareOptionsSelector", () => {
  const renderComponent = (props = {}) => {
    return createCustomRenderer({ context: {} })(
      <SoftwareOptionsSelector {...defaultProps} {...props} />
    );
  };

  it("calls onToggleSelfService when the self-service slider is toggled", () => {
    const onToggleSelfService = jest.fn();
    renderComponent({ onToggleSelfService });

    const selfServiceSwitch = getSwitchByLabelText("Self-service");
    fireEvent.click(selfServiceSwitch);

    expect(onToggleSelfService).toHaveBeenCalledTimes(1);
    // Slider calls onChange with no args
    expect(onToggleSelfService).toHaveBeenCalledWith();
  });

  it("calls onToggleAutomaticInstall when the automatic install slider is toggled", () => {
    const onToggleAutomaticInstall = jest.fn();
    renderComponent({ onToggleAutomaticInstall });

    const automaticInstallSwitch = getSwitchByLabelText("Automatic install");
    fireEvent.click(automaticInstallSwitch);

    expect(onToggleAutomaticInstall).toHaveBeenCalledTimes(1);
    expect(onToggleAutomaticInstall).toHaveBeenCalledWith();
  });

  it("enables self-service and disables automatic install sliders for iOS", () => {
    renderComponent({ platform: "ios" });

    const selfServiceSwitch = getSwitchByLabelText("Self-service");
    const automaticInstallSwitch = getSwitchByLabelText("Automatic install");

    expect(selfServiceSwitch.disabled).toBe(false);
    expect(automaticInstallSwitch.disabled).toBe(true);
  });

  it("enables self-service and disables automatic install sliders for iPadOS", () => {
    renderComponent({ platform: "ipados" });

    const selfServiceSwitch = getSwitchByLabelText("Self-service");
    const automaticInstallSwitch = getSwitchByLabelText("Automatic install");

    expect(selfServiceSwitch.disabled).toBe(false);
    expect(automaticInstallSwitch.disabled).toBe(true);
  });

  it("disables sliders when disableOptions is true", () => {
    renderComponent({ disableOptions: true });

    const selfServiceSwitch = getSwitchByLabelText("Self-service");
    const automaticInstallSwitch = getSwitchByLabelText("Automatic install");

    expect(selfServiceSwitch.disabled).toBe(true);
    expect(automaticInstallSwitch.disabled).toBe(true);
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

  it("does not render automatic install slider when isEditingSoftware is true", () => {
    renderComponent({ isEditingSoftware: true });

    expect(screen.queryByText("Automatic install")).not.toBeInTheDocument();
  });

  it("displays platform-specific message for iOS", () => {
    renderComponent({ platform: "ios" });

    expect(
      screen.getByText(/Automatic install for iOS and iPadOS is coming soon./i)
    ).toBeInTheDocument();
  });

  it("displays platform-specific message for iPadOS", () => {
    renderComponent({ platform: "ipados" });

    expect(
      screen.getByText(/Automatic install for iOS and iPadOS is coming soon./i)
    ).toBeInTheDocument();
  });

  it("does not render automatic install slider in edit mode", () => {
    renderComponent({ isEditingSoftware: true });

    expect(screen.queryByText("Automatic install")).not.toBeInTheDocument();
  });
});
