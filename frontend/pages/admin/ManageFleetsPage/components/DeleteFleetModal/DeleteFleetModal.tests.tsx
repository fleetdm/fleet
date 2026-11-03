import { screen } from "@testing-library/react";
import React from "react";

import { renderWithSetup } from "test/test-utils";

import DeleteFleetModal from "./DeleteFleetModal";

describe("DeleteFleetModal", () => {
  const defaultProps = {
    name: "Workstations",
    isUpdatingFleets: false,
    onSubmit: jest.fn(),
    onCancel: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("calls onSubmit once when Delete is clicked", async () => {
    const { user } = renderWithSetup(<DeleteFleetModal {...defaultProps} />);

    await user.click(screen.getByRole("button", { name: "Delete" }));

    expect(defaultProps.onSubmit).toHaveBeenCalledTimes(1);
  });

  it("calls onSubmit once when Enter is pressed", async () => {
    const { user } = renderWithSetup(<DeleteFleetModal {...defaultProps} />);

    await user.keyboard("{Enter}");

    expect(defaultProps.onSubmit).toHaveBeenCalledTimes(1);
  });

  it("does not call onSubmit again on a second click while the fleet is being deleted", async () => {
    const { user, rerender } = renderWithSetup(
      <DeleteFleetModal {...defaultProps} />
    );

    await user.click(screen.getByRole("button", { name: "Delete" }));
    rerender(<DeleteFleetModal {...defaultProps} isUpdatingFleets />);
    await user.click(screen.getByRole("button", { name: "Delete" }));

    expect(defaultProps.onSubmit).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: "Delete" })).toBeDisabled();
  });

  it("does not call onSubmit again on a second Enter while the fleet is being deleted", async () => {
    const { user, rerender } = renderWithSetup(
      <DeleteFleetModal {...defaultProps} />
    );

    await user.keyboard("{Enter}");
    rerender(<DeleteFleetModal {...defaultProps} isUpdatingFleets />);
    await user.keyboard("{Enter}");

    expect(defaultProps.onSubmit).toHaveBeenCalledTimes(1);
  });
});
