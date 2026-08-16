import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
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
  it("calls onToggleDeploySoftware when the deploy software slider is toggled", () => {
    const onToggleDeploySoftware = jest.fn();
    render(
      <SoftwareDeploySlider
        {...defaultProps}
        onToggleDeploySoftware={onToggleDeploySoftware}
      />
    );

    const deploySoftwareSwitch = getSwitchByLabelText("Deploy");
    fireEvent.click(deploySoftwareSwitch);

    expect(onToggleDeploySoftware).toHaveBeenCalledTimes(1);
    expect(onToggleDeploySoftware).toHaveBeenCalledWith();
  });
});
