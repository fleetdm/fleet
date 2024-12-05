import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import LoginForm from "./LoginForm";

const [validEmail, invalidEmail] = ["hi@thegnar.co", "invalid-email"];
const password = "p@ssw0rd";

describe("LoginForm - component", () => {
  const settings = { sso_enabled: false };
  const submitSpy = jest.fn();
  const baseError = "Unable to authenticate the current user";

  it("renders the base error", () => {
    render(
      <LoginForm
        baseError={baseError}
        handleSubmit={submitSpy}
        isSubmitting={false}
        pendingEmail={false}
        ssoSettings={settings}
      />
    );

    expect(screen.getByText(baseError)).toBeInTheDocument();
  });

  it("should not render the base error", () => {
    render(
      <LoginForm
        handleSubmit={submitSpy}
        isSubmitting={false}
        pendingEmail={false}
        ssoSettings={settings}
      />
    );

    expect(screen.queryByText(baseError)).not.toBeInTheDocument();
  });

  it("renders 2 InputField components", () => {
    render(
      <LoginForm
        handleSubmit={submitSpy}
        isSubmitting={false}
        pendingEmail={false}
        ssoSettings={settings}
      />
    );

    expect(screen.getByPlaceholderText("Email")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Password")).toBeInTheDocument();
  });

  it("rejects an empty or invalid email field without submitting", async () => {
    const { user } = renderWithSetup(
      <LoginForm
        handleSubmit={submitSpy}
        isSubmitting={false}
        pendingEmail={false}
        ssoSettings={settings}
      />
    );

    // enter a valid password
    await user.type(screen.getByPlaceholderText("Password"), password);

    // try to log in
    await user.click(screen.getByRole("button", { name: "Log in" }));
    expect(
      screen.getByText("Email field must be completed")
    ).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();

    // enter an invalid email
    await user.type(screen.getByPlaceholderText("Email"), invalidEmail);

    // try to log in again
    await user.click(screen.getByRole("button", { name: "Log in" }));
    expect(
      screen.getByText("Email must be a valid email address")
    ).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("rejects an empty password field without submitting", async () => {
    const { user } = renderWithSetup(
      <LoginForm
        handleSubmit={submitSpy}
        isSubmitting={false}
        pendingEmail={false}
        ssoSettings={settings}
      />
    );

    await user.type(screen.getByPlaceholderText("Email"), validEmail);

    // try to log in without entering a password
    await user.click(screen.getByRole("button", { name: "Log in" }));

    expect(
      screen.getByText("Password field must be completed")
    ).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("does not submit the form when both fields are empty", async () => {
    const { user } = renderWithSetup(
      <LoginForm
        handleSubmit={submitSpy}
        isSubmitting={false}
        pendingEmail={false}
        ssoSettings={settings}
      />
    );

    await user.click(screen.getByRole("button", { name: "Log in" }));

    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("submits the form data when valid form data is submitted", async () => {
    const { user } = renderWithSetup(
      <LoginForm
        handleSubmit={submitSpy}
        isSubmitting={false}
        pendingEmail={false}
        ssoSettings={settings}
      />
    );

    await user.type(screen.getByPlaceholderText("Email"), validEmail);
    await user.type(screen.getByPlaceholderText("Password"), password);
    await user.click(screen.getByRole("button", { name: "Log in" }));

    expect(submitSpy).toHaveBeenCalledWith({
      email: validEmail,
      password,
    });
  });
});
