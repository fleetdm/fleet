import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

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

  it("it does not submit the form when the form fields have not been filled out", () => {
    render(<LoginForm handleSubmit={submitSpy} ssoSettings={settings} />);

    // when
    fireEvent.click(screen.getByRole("button", { name: "Login" }));

    // then
    expect(submitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Email field must be completed")
    ).toBeInTheDocument();
  });

  it("submits the form data when form is submitted", () => {
    render(<LoginForm handleSubmit={submitSpy} ssoSettings={settings} />);

    // when
    userEvent.type(
      screen.getByRole("textbox", { name: "Email" }),
      "my@email.com"
    );
    userEvent.type(screen.getByPlaceholderText("Password"), "p@ssw0rd");
    fireEvent.click(screen.getByRole("button", { name: "Login" }));

    // then
    expect(submitSpy).toHaveBeenCalledWith({
      email: "my@email.com",
      password: "p@ssw0rd",
    });
  });
});
