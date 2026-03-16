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

  it("enables self-service sliders for iOS", () => {
    renderComponent({ platform: "ios" });

    const selfServiceSwitch = getSwitchByLabelText("Self-service");
    expect(selfServiceSwitch.disabled).toBe(false);
  });

  it("enables self-service  for iPadOS", () => {
    renderComponent({ platform: "ipados" });

    const selfServiceSwitch = getSwitchByLabelText("Self-service");
    expect(selfServiceSwitch.disabled).toBe(false);
  });

  it("disables self-service when disableOptions is true", () => {
    renderComponent({ disableOptions: true });

    const selfServiceSwitch = getSwitchByLabelText("Self-service");

    expect(selfServiceSwitch.disabled).toBe(true);
  });
});
