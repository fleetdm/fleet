import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import ForgotPasswordForm from "./ForgotPasswordForm";

const [validEmail, invalidEmail] = ["hi@thegnar.co", "invalid-email"];

describe("ForgotPasswordForm - component", () => {
  const handleSubmit = jest.fn();
  it("renders correctly", () => {
    render(<ForgotPasswordForm handleSubmit={handleSubmit} />);

    expect(screen.getByRole("textbox", { name: /email/i })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Get instructions" })
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

  it("correctly validates the email field", async () => {
    const { user } = renderWithSetup(
      <ForgotPasswordForm handleSubmit={handleSubmit} />
    );

    await user.click(screen.getByRole("button", { name: "Get instructions" }));
    let emailError = screen.getByText("Email field must be completed");
    expect(emailError).toBeInTheDocument();
    expect(handleSubmit).not.toHaveBeenCalled();

    await user.type(screen.getByPlaceholderText("Email"), invalidEmail);
    await user.click(screen.getByRole("button", { name: "Get instructions" }));

    emailError = screen.getByText("Email must be a valid email address");
    expect(emailError).toBeInTheDocument();
    expect(handleSubmit).not.toHaveBeenCalled();
  });

  it("submits the form data when the form is submitted", async () => {
    const { user } = renderWithSetup(
      <ForgotPasswordForm handleSubmit={handleSubmit} />
    );

    await user.type(screen.getByPlaceholderText("Email"), validEmail);

    await user.click(screen.getByRole("button", { name: "Get instructions" }));
    expect(handleSubmit).toHaveBeenCalledWith({ email: validEmail });
  });
});
