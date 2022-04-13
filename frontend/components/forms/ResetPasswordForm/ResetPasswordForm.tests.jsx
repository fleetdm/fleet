import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import ResetPasswordForm from "./ResetPasswordForm";

describe("ResetPasswordForm - component", () => {
  const newPassword = "p@ssw0rd";
  const submitSpy = jest.fn();
  it("renders correctly", () => {
    render(<ResetPasswordForm handleSubmit={submitSpy} />);

    expect(screen.getByLabelText("New password")).toBeInTheDocument();
    expect(screen.getByLabelText("Confirm password")).toBeInTheDocument();

    expect(
      screen.getByRole("button", { name: "Reset password" })
    ).toBeInTheDocument();
  });

  it("it does not submit the form when the new form fields have not been filled out", async () => {
    render(<ResetPasswordForm handleSubmit={submitSpy} />);

    // when
    fireEvent.click(screen.getByRole("button", { name: "Reset password" }));
    // then
    expect(
      await screen.findByText("New password field must be completed")
    ).toBeInTheDocument();
    expect(
      await screen.findByText(
        "New password confirmation field must be completed"
      )
    ).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("it does not submit the form when only the new password field has been filled out", async () => {
    render(<ResetPasswordForm handleSubmit={submitSpy} />);

    // when
    userEvent.type(screen.getByLabelText("New password"), newPassword);
    fireEvent.click(screen.getByRole("button", { name: "Reset password" }));
    // then
    expect(
      await screen.findByText(
        "New password confirmation field must be completed"
      )
    ).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("it does not submit the form when only the new confirmation password field has been filled out", async () => {
    render(<ResetPasswordForm handleSubmit={submitSpy} />);

    // when
    userEvent.type(screen.getByLabelText("Confirm password"), newPassword);
    fireEvent.click(screen.getByRole("button", { name: "Reset password" }));
    // then
    expect(
      await screen.findByText("New password field must be completed")
    ).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("does not submit the form if the new password confirmation does not match", async () => {
    render(<ResetPasswordForm handleSubmit={submitSpy} />);

    // when
    userEvent.type(screen.getByLabelText("New password"), newPassword);
    userEvent.type(
      screen.getByLabelText("Confirm password"),
      "not my new password"
    );
    fireEvent.click(screen.getByRole("button", { name: "Reset password" }));
    // then
    expect(
      await screen.findByText("Passwords do not match")
    ).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("does not submit the form if the password is invalid", async () => {
    const invalidPassword = "invalid";

    render(<ResetPasswordForm handleSubmit={submitSpy} />);

    // when
    userEvent.type(screen.getByLabelText("New password"), invalidPassword);
    userEvent.type(screen.getByLabelText("Confirm password"), invalidPassword);
    fireEvent.click(screen.getByRole("button", { name: "Reset password" }));
    // then
    expect(
      await screen.findByText("Password must meet the criteria below")
    ).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("submits the form data when the form is submitted", async () => {
    render(<ResetPasswordForm handleSubmit={submitSpy} />);

    // when
    userEvent.type(screen.getByLabelText("New password"), newPassword);
    userEvent.type(screen.getByLabelText("Confirm password"), newPassword);
    fireEvent.click(screen.getByRole("button", { name: "Reset password" }));
    // then
    expect(submitSpy).toHaveBeenCalledWith({
      new_password: newPassword,
      new_password_confirmation: newPassword,
    });
  });
});
