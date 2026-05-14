import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import userEvent from "@testing-library/user-event";

import RenameFleetModal from "./RenameFleetModal";

describe("RenameFleetModal", () => {
  const defaultProps = {
    onCancel: jest.fn(),
    onSubmit: jest.fn(),
    defaultName: "Existing Fleet Name",
    backendValidators: {},
    isUpdatingFleets: false,
  };

  const render = createCustomRenderer();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders the modal with the save button enabled for the default name", () => {
    render(<RenameFleetModal {...defaultProps} />);

    const saveButton = screen.getByRole("button", { name: "Save" });
    expect(saveButton).toBeEnabled();
  });

  it("disables the save button when the name is cleared", async () => {
    render(<RenameFleetModal {...defaultProps} />);

    const nameInput = screen.getByLabelText("Fleet name");
    await userEvent.clear(nameInput);

    const saveButton = screen.getByRole("button", { name: "Save" });
    expect(saveButton).toBeDisabled();
  });

  it("disables the save button when only spaces are entered", async () => {
    render(<RenameFleetModal {...defaultProps} />);

    const nameInput = screen.getByLabelText("Fleet name");
    await userEvent.clear(nameInput);
    await userEvent.type(nameInput, "     ");

    const saveButton = screen.getByRole("button", { name: "Save" });
    expect(saveButton).toBeDisabled();
  });

  it("calls onSubmit with trimmed name", async () => {
    render(<RenameFleetModal {...defaultProps} />);

    const nameInput = screen.getByLabelText("Fleet name");
    await userEvent.clear(nameInput);
    await userEvent.type(nameInput, "  New Name  ");

    const saveButton = screen.getByRole("button", { name: "Save" });
    await userEvent.click(saveButton);

    expect(defaultProps.onSubmit).toHaveBeenCalledWith({ name: "New Name" });
  });

  it("does not call onSubmit when name is whitespace-only", async () => {
    render(<RenameFleetModal {...defaultProps} />);

    const nameInput = screen.getByLabelText("Fleet name");
    await userEvent.clear(nameInput);
    await userEvent.type(nameInput, "     ");

    const saveButton = screen.getByRole("button", { name: "Save" });
    expect(saveButton).toBeDisabled();
    expect(defaultProps.onSubmit).not.toHaveBeenCalled();
  });
});
