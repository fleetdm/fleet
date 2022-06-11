import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import ForgotPasswordForm from "./ForgotPasswordForm";

const email = "hi@thegnar.co";

describe("ForgotPasswordForm - component", () => {
  const handleSubmit = jest.fn();
  it("renders correctly", () => {
    render(<ForgotPasswordForm handleSubmit={handleSubmit} />);

    expect(screen.getByRole("textbox", { name: /email/i })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Reset password" })
    ).toBeInTheDocument();
  });

  it("should renders the component with the server error prop", () => {
    const baseError = "Can't find the specified user";
    // when
    render(
      <ForgotPasswordForm
        handleSubmit={handleSubmit}
        serverErrors={{ base: baseError }}
      />
    );
    // then
    expect(
      screen.getByText("Can't find the specified user")
    ).toBeInTheDocument();
  });

  it("should test validation for email field", async () => {
    render(<ForgotPasswordForm handleSubmit={handleSubmit} />);

    // when
    fireEvent.click(screen.getByRole("button", { name: "Reset password" }));
    // then
    expect(
      await screen.findByText("Email field must be completed")
    ).toBeInTheDocument();
    expect(handleSubmit).not.toHaveBeenCalled();

    // when
    userEvent.type(screen.getByPlaceholderText("Email"), "invalid-email");
    fireEvent.click(screen.getByRole("button", { name: "Reset password" }));
    // then
    expect(
      await screen.findByText("invalid-email is not a valid email")
    ).toBeInTheDocument();
    expect(handleSubmit).not.toHaveBeenCalled();
  });

  it("submits the form data when the form is submitted", () => {
    render(<ForgotPasswordForm handleSubmit={handleSubmit} />);

    // when
    userEvent.type(screen.getByPlaceholderText("Email"), email);
    fireEvent.click(screen.getByRole("button", { name: "Reset password" }));
    // then
    expect(handleSubmit).toHaveBeenCalledWith({ email });
  });
});
