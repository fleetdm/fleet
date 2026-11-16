import { render, screen } from "@testing-library/react";
import React from "react";

import { renderWithSetup } from "test/test-utils";

import ChangePasswordForm from "./ChangePasswordForm";

describe("ChangePasswordForm - component", () => {
  const props = {
    handleSubmit: jest.fn(),
    onCancel: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders the correct fields and buttons", () => {
    render(<ChangePasswordForm {...props} />);

    expect(screen.getByLabelText("Current password")).toBeInTheDocument();
    expect(screen.getByLabelText("New password")).toBeInTheDocument();
    expect(screen.getByLabelText("Confirm password")).toBeInTheDocument();

    expect(
      screen.getByRole("button", { name: "Change password" })
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("renders the password fields as HTML password fields", () => {
    render(<ChangePasswordForm {...props} />);

    expect(screen.getByLabelText("Current password")).toHaveAttribute(
      "type",
      "password"
    );
    expect(screen.getByLabelText("New password")).toHaveAttribute(
      "type",
      "password"
    );
    expect(screen.getByLabelText("Confirm password")).toHaveAttribute(
      "type",
      "password"
    );
  });

  it("surfaces every required-field error on submit", async () => {
    const { user } = renderWithSetup(<ChangePasswordForm {...props} />);

    await user.click(screen.getByRole("button", { name: "Change password" }));

    expect(
      await screen.findByText("Enter your current password")
    ).toBeInTheDocument();
    expect(screen.getByText("Enter a new password")).toBeInTheDocument();
    expect(screen.getByText("Confirm your new password")).toBeInTheDocument();
    expect(props.handleSubmit).not.toHaveBeenCalled();
  });

  it("flags a too-short new password on submit even when other fields are empty", async () => {
    const { user } = renderWithSetup(<ChangePasswordForm {...props} />);

    await user.type(screen.getByLabelText("New password"), "short1!");
    await user.click(screen.getByRole("button", { name: "Change password" }));

    expect(
      screen.getByText("Enter a password with at least 12 characters")
    ).toBeInTheDocument();
    expect(props.handleSubmit).not.toHaveBeenCalled();
  });

  it("validates the new password on blur", async () => {
    const { user } = renderWithSetup(<ChangePasswordForm {...props} />);

    await user.type(screen.getByLabelText("New password"), "short1!");
    await user.tab();

    expect(
      await screen.findByText("Enter a password with at least 12 characters")
    ).toBeInTheDocument();
  });

  it("flags mismatched confirmation on submit", async () => {
    const { user } = renderWithSetup(<ChangePasswordForm {...props} />);

    await user.type(screen.getByLabelText("Current password"), "old-pass-1!");
    await user.type(screen.getByLabelText("New password"), "password123!");
    await user.type(screen.getByLabelText("Confirm password"), "password123?");
    await user.click(screen.getByRole("button", { name: "Change password" }));

    expect(screen.getByText("Match the password above")).toBeInTheDocument();
    expect(props.handleSubmit).not.toHaveBeenCalled();
  });

  it("clears an error when the field is focused again", async () => {
    const { container, user } = renderWithSetup(
      <ChangePasswordForm {...props} />
    );

    await user.click(screen.getByRole("button", { name: "Change password" }));
    expect(
      await screen.findByText("Enter your current password")
    ).toBeInTheDocument();

    const currentPasswordInput = container.querySelector<HTMLInputElement>(
      'input[name="old_password"]'
    );
    expect(currentPasswordInput).not.toBeNull();
    await user.click(currentPasswordInput!);

    expect(
      screen.queryByText("Enter your current password")
    ).not.toBeInTheDocument();
  });

  it("calls handleSubmit with form data when valid", async () => {
    const { user } = renderWithSetup(<ChangePasswordForm {...props} />);

    const expectedFormData = {
      old_password: "password123#",
      new_password: "password123!",
      new_password_confirmation: "password123!",
    };

    await user.type(
      screen.getByLabelText("Current password"),
      expectedFormData.old_password
    );
    await user.type(
      screen.getByLabelText("New password"),
      expectedFormData.new_password
    );
    await user.type(
      screen.getByLabelText("Confirm password"),
      expectedFormData.new_password_confirmation
    );
    await user.click(screen.getByRole("button", { name: "Change password" }));

    expect(props.handleSubmit).toHaveBeenCalledWith(expectedFormData);
  });

  it("calls onCancel when Cancel is clicked", async () => {
    const { user } = renderWithSetup(<ChangePasswordForm {...props} />);

    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(props.onCancel).toHaveBeenCalled();
  });
});
