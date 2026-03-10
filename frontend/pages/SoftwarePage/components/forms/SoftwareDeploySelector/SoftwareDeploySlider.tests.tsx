import React from "react";
import { screen, fireEvent } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import SoftwareDeploySlider from "./SoftwareDeploySlider";

const defaultProps = {
  deploySoftware: false,
  onToggleDeploySoftware: jest.fn(),
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
      <SoftwareDeploySlider {...defaultProps} {...props} />
    );
  };

  it("calls onToggleDeploySoftware when the deploy software slider is toggled", () => {
    const onToggleDeploySoftware = jest.fn();
    renderComponent({ onToggleDeploySoftware });

    const deploySoftwareSwitch = getSwitchByLabelText("Deploy");
    fireEvent.click(deploySoftwareSwitch);

    expect(onToggleDeploySoftware).toHaveBeenCalledTimes(1);
    expect(onToggleDeploySoftware).toHaveBeenCalledWith();
  });

  it("disables deploy software slider for iOS", () => {
    renderComponent({ platform: "ios" });

    const deploySoftwareSwitch = getSwitchByLabelText("Deploy");

    expect(deploySoftwareSwitch.disabled).toBe(true);
  });

  it("disables deploy software slider for iPadOS", () => {
    renderComponent({ platform: "ipados" });

    const deploySoftwareSwitch = getSwitchByLabelText("Deploy");

    expect(deploySoftwareSwitch.disabled).toBe(true);
  });

  it("disables slider when disableOptions is true", () => {
    renderComponent({ disableOptions: true });

    const deploySoftwareSwitch = getSwitchByLabelText("Deploy");

    expect(deploySoftwareSwitch.disabled).toBe(true);
  });

  it("renders the InfoBanner when deploySoftware is true and isCustomPackage is true", () => {
    renderComponent({
      formData: { deploySoftware: true },
      isCustomPackage: true,
    });

    expect(
      screen.getByText(
        /Installing software over existing installations might cause issues/i
      )
    ).toBeInTheDocument();
  });

  it("does not render the InfoBanner when deploySoftware is false", () => {
    renderComponent({
      formData: { deploySoftware: false },
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
      formData: { deploySoftware: true },
      isCustomPackage: false,
    });

    expect(
      screen.queryByText(
        /Installing software over existing installations might cause issues/i
      )
    ).not.toBeInTheDocument();
  });

  it("displays platform-specific message for iOS", () => {
    renderComponent({ platform: "ios" });

    expect(
      screen.getByText(/Deploy software for iOS and iPadOS is coming soon./i)
    ).toBeInTheDocument();
  });

  it("displays platform-specific message for iPadOS", () => {
    renderComponent({ platform: "ipados" });

    expect(
      screen.getByText(/Deploy software for iOS and iPadOS is coming soon./i)
    ).toBeInTheDocument();
  });
});
