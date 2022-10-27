import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/testingUtils";

import ForgotPasswordForm from "./ForgotPasswordForm";

const email = "hi@thegnar.co";

describe("ForgotPasswordForm - component", () => {
  const handleSubmit = jest.fn();
  it("renders correctly", () => {
    render(<ForgotPasswordForm handleSubmit={handleSubmit} />);

    expect(screen.getByRole("textbox", { name: /email/i })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Send email" })
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
    const { user } = renderWithSetup(
      <ForgotPasswordForm handleSubmit={handleSubmit} />
    );

    await user.click(screen.getByRole("button", { name: "Send email" }));
    let emailError = screen.getByText("Email field must be completed");
    expect(emailError).toBeInTheDocument();
    expect(handleSubmit).not.toHaveBeenCalled();

    await user.type(screen.getByPlaceholderText("Email"), "invalid-email");
    await user.click(screen.getByRole("button", { name: "Send email" }));

    emailError = screen.getByText("invalid-email is not a valid email");
    expect(emailError).toBeInTheDocument();
    expect(handleSubmit).not.toHaveBeenCalled();
  });

  it("submits the form data when the form is submitted", async () => {
    const { user } = renderWithSetup(
      <ForgotPasswordForm handleSubmit={handleSubmit} />
    );

    await user.type(screen.getByPlaceholderText("Email"), email);
    await user.click(screen.getByRole("button", { name: "Send email" }));

    expect(handleSubmit).toHaveBeenCalledWith({ email });
  });
});
