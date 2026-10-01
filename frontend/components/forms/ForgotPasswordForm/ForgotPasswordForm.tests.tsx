import { render, screen } from "@testing-library/react";
import React from "react";

import { renderWithSetup } from "test/test-utils";

import ForgotPasswordForm from "./ForgotPasswordForm";

const [validEmail, invalidEmail] = ["hi@thegnar.co", "invalid-email"];

describe("ForgotPasswordForm - component", () => {
  const handleSubmit = jest.fn();

  it("renders the email input and submit button", () => {
    render(<ForgotPasswordForm handleSubmit={handleSubmit} />);

    expect(screen.getByPlaceholderText("Email")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Get instructions" })
    ).toBeInTheDocument();
  });

  it("rejects an empty or invalid email without submitting", async () => {
    const { user } = renderWithSetup(
      <ForgotPasswordForm handleSubmit={handleSubmit} />
    );

    await user.click(screen.getByRole("button", { name: "Get instructions" }));
    expect(screen.getByText("Enter your email")).toBeInTheDocument();
    expect(handleSubmit).not.toHaveBeenCalled();

    await user.type(screen.getByPlaceholderText("Email"), invalidEmail);
    await user.click(screen.getByRole("button", { name: "Get instructions" }));

    expect(screen.getByText("Enter a valid email")).toBeInTheDocument();
    expect(handleSubmit).not.toHaveBeenCalled();
  });

  it("submits the form data when valid form data is submitted", async () => {
    const { user } = renderWithSetup(
      <ForgotPasswordForm handleSubmit={handleSubmit} />
    );

    await user.type(screen.getByPlaceholderText("Email"), validEmail);
    await user.click(screen.getByRole("button", { name: "Get instructions" }));

    expect(handleSubmit).toHaveBeenCalledWith({ email: validEmail });
  });
});
