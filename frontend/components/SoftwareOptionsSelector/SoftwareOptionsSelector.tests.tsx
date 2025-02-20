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
  },
  onToggleAutomaticInstall: jest.fn(),
  onToggleSelfService: jest.fn(),
};

describe("SoftwareOptionsSelector", () => {
  const renderComponent = (props = {}) => {
    return createCustomRenderer({ context: {} })(
      <SoftwareOptionsSelector {...defaultProps} {...props} />
    );
  };

  it("renders the self-service checkbox with the correct initial value", () => {
    const { rerender } = renderComponent({
      formData: { ...defaultProps.formData, selfService: true },
    });

    expect(screen.getByLabelText("Self-service")).toBeChecked();

    rerender(<SoftwareOptionsSelector {...defaultProps} />);

    expect(screen.getByLabelText("Self-service")).not.toBeChecked();
  });

  it("calls onToggleSelfService when the self-service checkbox is toggled", () => {
    const onToggleSelfService = jest.fn();
    renderComponent({ onToggleSelfService });

    fireEvent.click(screen.getByLabelText("Self-service"));

    expect(onToggleSelfService).toHaveBeenCalledTimes(1);
    expect(onToggleSelfService).toHaveBeenCalledWith(true);
  });

  it("renders the automatic install checkbox with the correct initial value", () => {
    const { rerender } = renderComponent({
      formData: { ...defaultProps.formData, automaticInstall: true },
    });

    expect(screen.getByLabelText("Automatic install")).toBeChecked();

    rerender(<SoftwareOptionsSelector {...defaultProps} />);

    expect(screen.getByLabelText("Automatic install")).not.toBeChecked();
  });

  it("calls onToggleAutomaticInstall when the automatic install checkbox is toggled", () => {
    const onToggleAutomaticInstall = jest.fn();
    renderComponent({ onToggleAutomaticInstall });

    fireEvent.click(screen.getByLabelText("Automatic install"));

    expect(onToggleAutomaticInstall).toHaveBeenCalledTimes(1);
    expect(onToggleAutomaticInstall).toHaveBeenCalledWith(true);
  });

  it("disables self-service and automatic install checkboxes for iOS and iPadOS", () => {
    renderComponent({ platform: "ios" });

    expect(screen.getByLabelText("Self-service")).toBeDisabled();
    expect(screen.getByLabelText("Automatic install")).toBeDisabled();

    renderComponent({ platform: "ipados" });

    expect(screen.getByLabelText("Self-service")).toBeDisabled();
    expect(screen.getByLabelText("Automatic install")).toBeDisabled();
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
});
