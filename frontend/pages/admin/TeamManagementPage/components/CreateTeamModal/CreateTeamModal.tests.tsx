import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import userEvent from "@testing-library/user-event";

import CreateTeamModal from "./CreateTeamModal";

describe("CreateTeamModal", () => {
  const defaultProps = {
    onCancel: jest.fn(),
    onSubmit: jest.fn(),
    backendValidators: {},
    isUpdatingTeams: false,
  };

  const render = createCustomRenderer();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders the modal with the create button disabled initially", () => {
    render(<CreateTeamModal {...defaultProps} />);

    const createButton = screen.getByRole("button", { name: "Create" });
    expect(createButton).toBeDisabled();
  });

  it("enables the create button when a valid name is entered", async () => {
    render(<CreateTeamModal {...defaultProps} />);

    const nameInput = screen.getByLabelText("Fleet name");
    await userEvent.type(nameInput, "Engineering");

    const createButton = screen.getByRole("button", { name: "Create" });
    expect(createButton).toBeEnabled();
  });

  it("keeps the create button disabled when only spaces are entered", async () => {
    render(<CreateTeamModal {...defaultProps} />);

    const nameInput = screen.getByLabelText("Fleet name");
    await userEvent.type(nameInput, "     ");

    const createButton = screen.getByRole("button", { name: "Create" });
    expect(createButton).toBeDisabled();
  });

  it("keeps the create button disabled when only tabs are entered", async () => {
    render(<CreateTeamModal {...defaultProps} />);

    const nameInput = screen.getByLabelText("Fleet name");
    await userEvent.type(nameInput, "\t\t\t");

    const createButton = screen.getByRole("button", { name: "Create" });
    expect(createButton).toBeDisabled();
  });

  it("calls onSubmit with trimmed name when a valid name is submitted", async () => {
    render(<CreateTeamModal {...defaultProps} />);

    const nameInput = screen.getByLabelText("Fleet name");
    await userEvent.type(nameInput, "  Engineering  ");

    const createButton = screen.getByRole("button", { name: "Create" });
    await userEvent.click(createButton);

    expect(defaultProps.onSubmit).toHaveBeenCalledWith({
      name: "Engineering",
    });
  });

  it("displays backend validation errors", () => {
    const props = {
      ...defaultProps,
      backendValidators: { name: "Team name already exists" },
    };
    render(<CreateTeamModal {...props} />);

    expect(screen.getByText("Team name already exists")).toBeInTheDocument();
  });

  it("clears errors when user types in the input", async () => {
    const props = {
      ...defaultProps,
      backendValidators: { name: "Team name already exists" },
    };
    render(<CreateTeamModal {...props} />);

    expect(screen.getByText("Team name already exists")).toBeInTheDocument();

    // When error is shown, the label text changes to the error message
    const nameInput = screen.getByRole("textbox");
    await userEvent.type(nameInput, "a");

    await waitFor(() => {
      expect(
        screen.queryByText("Team name already exists")
      ).not.toBeInTheDocument();
    });
  });
});
