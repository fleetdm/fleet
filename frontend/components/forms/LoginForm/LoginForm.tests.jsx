import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/testingUtils";

import LoginForm from "./LoginForm";

describe("LoginForm - component", () => {
  const settings = { sso_enabled: false };
  const submitSpy = jest.fn();
  const baseError = "Unable to authenticate the current user";

  it("renders the base error", () => {
    render(
      <LoginForm
        serverErrors={{ base: baseError }}
        handleSubmit={submitSpy}
        ssoSettings={settings}
      />
    );

    expect(screen.getByText(baseError)).toBeInTheDocument();
  });

  it("should not render the base error", () => {
    render(<LoginForm handleSubmit={submitSpy} ssoSettings={settings} />);

    expect(screen.queryByText(baseError)).not.toBeInTheDocument();
  });

  it("renders 2 InputField components", () => {
    render(<LoginForm handleSubmit={submitSpy} ssoSettings={settings} />);

    expect(screen.getByRole("textbox", { name: "Email" })).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Password")).toBeInTheDocument();
  });

  it("it does not submit the form when the form fields have not been filled out", async () => {
    const { user } = renderWithSetup(
      <LoginForm handleSubmit={submitSpy} ssoSettings={settings} />
    );

    await user.click(screen.getByRole("button", { name: "Login" }));

    expect(submitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Email field must be completed")
    ).toBeInTheDocument();
  });

  it("submits the form data when form is submitted", async () => {
    const { user } = renderWithSetup(
      <LoginForm handleSubmit={submitSpy} ssoSettings={settings} />
    );

    await user.type(
      screen.getByRole("textbox", { name: "Email" }),
      "my@email.com"
    );
    await user.type(screen.getByPlaceholderText("Password"), "p@ssw0rd");
    await user.click(screen.getByRole("button", { name: "Login" }));

    expect(submitSpy).toHaveBeenCalledWith({
      email: "my@email.com",
      password: "p@ssw0rd",
    });
  });
});
