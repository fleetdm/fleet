import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import ResetPasswordForm from "./ResetPasswordForm";

describe("ResetPasswordForm - component", () => {
  const newPassword = "password123!";
  const submitSpy = jest.fn();
  it("renders correctly", () => {
    render(<ResetPasswordForm handleSubmit={submitSpy} />);

    expect(screen.getByPlaceholderText("New password")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Confirm password")).toBeInTheDocument();

    expect(
      screen.getByRole("button", { name: "Reset password" })
    ).toBeInTheDocument();
  });

  it("it does not submit the form when the new form fields have not been filled out", async () => {
    const { user } = renderWithSetup(
      <ResetPasswordForm handleSubmit={submitSpy} />
    );

    await user.click(screen.getByRole("button", { name: "Reset password" }));

    const passwordError = screen.getByText(
      "New password field must be completed"
    );
    const passwordConfirmError = screen.getByText(
      "New password confirmation field must be completed"
    );
    expect(passwordError).toBeInTheDocument();
    expect(passwordConfirmError).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("it does not submit the form when only the new password field has been filled out", async () => {
    const { user } = renderWithSetup(
      <ResetPasswordForm handleSubmit={submitSpy} />
    );

    await user.type(screen.getByPlaceholderText("New password"), newPassword);
    await user.click(screen.getByRole("button", { name: "Reset password" }));

    const passwordError = screen.getByText(
      "New password confirmation field must be completed"
    );

    expect(passwordError).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("it does not submit the form when only the new confirmation password field has been filled out", async () => {
    const { user } = renderWithSetup(
      <ResetPasswordForm handleSubmit={submitSpy} />
    );

    await user.type(
      screen.getByPlaceholderText("Confirm password"),
      newPassword
    );
    await user.click(screen.getByRole("button", { name: "Reset password" }));

    const passwordError = screen.getByText(
      "New password field must be completed"
    );
    expect(passwordError).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("does not submit the form if the new password confirmation does not match", async () => {
    const { user } = renderWithSetup(
      <ResetPasswordForm handleSubmit={submitSpy} />
    );

    await user.type(screen.getByPlaceholderText("New password"), newPassword);
    await user.type(
      screen.getByPlaceholderText("Confirm password"),
      "not my new password"
    );
    await user.click(screen.getByRole("button", { name: "Reset password" }));
    const passwordError = screen.getByText("Passwords do not match");

    expect(passwordError).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("does not submit the form if the password is invalid", async () => {
    const invalidPassword = "invalid";
    const { user } = renderWithSetup(
      <ResetPasswordForm handleSubmit={submitSpy} />
    );

    await user.type(
      screen.getByPlaceholderText("New password"),
      invalidPassword
    );
    await user.type(
      screen.getByPlaceholderText("Confirm password"),
      invalidPassword
    );
    await user.click(screen.getByRole("button", { name: "Reset password" }));

    const passwordError = screen.getByText(
      "Password must meet the criteria below"
    );
    expect(passwordError).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("submits the form data when the form is submitted", async () => {
    const { user } = renderWithSetup(
      <ResetPasswordForm handleSubmit={submitSpy} />
    );

    await user.type(screen.getByPlaceholderText("New password"), newPassword);
    await user.type(
      screen.getByPlaceholderText("Confirm password"),
      newPassword
    );
    await user.click(screen.getByRole("button", { name: "Reset password" }));

    expect(submitSpy).toHaveBeenCalledWith({
      new_password: newPassword,
      new_password_confirmation: newPassword,
    });
  });
});
