import { render, screen } from "@testing-library/react";
import React from "react";

import { renderWithSetup } from "test/test-utils";

import ChangeEmailForm from "./ChangeEmailForm";

describe("ChangeEmailForm - component", () => {
  const props = {
    handleSubmit: jest.fn(),
    onCancel: jest.fn(),
  };
  const password = "p@ssw0rd";

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders the component correctly", () => {
    render(<ChangeEmailForm {...props} />);

    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Submit" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("does not submit when the password field is empty", async () => {
    const { user } = renderWithSetup(<ChangeEmailForm {...props} />);

    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(await screen.findByText("Enter your password")).toBeInTheDocument();
    expect(props.handleSubmit).not.toHaveBeenCalled();
  });

  it("submits the password merged with defaultData", async () => {
    const { user } = renderWithSetup(
      <ChangeEmailForm {...props} defaultData={{ email: "new@example.com" }} />
    );

    await user.type(screen.getByLabelText("Password"), password);
    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(props.handleSubmit).toHaveBeenCalledWith({
      email: "new@example.com",
      password,
    });
  });

  it("calls onCancel when the Cancel button is pressed", async () => {
    const { user } = renderWithSetup(<ChangeEmailForm {...props} />);

    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(props.onCancel).toHaveBeenCalled();
  });
});
