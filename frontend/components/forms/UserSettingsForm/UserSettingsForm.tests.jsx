import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import UserSettingsForm from "components/forms/UserSettingsForm";

describe("UserSettingsForm - component", () => {
  const defaultProps = {
    handleSubmit: jest.fn(),
    onCancel: jest.fn(),
  };

  it("renders correctly", () => {
    render(<UserSettingsForm {...defaultProps} />);

    expect(
      screen.getByRole("textbox", { name: /email \(required\)/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("textbox", { name: /full name \(required\)/i })
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Update" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("should pass validation checks for input fields", async () => {
    render(<UserSettingsForm {...defaultProps} />);

    // when
    fireEvent.click(screen.getByRole("button", { name: "Update" }));
    // then
    expect(defaultProps.handleSubmit).not.toHaveBeenCalled();
    expect(
      await screen.findByText("Email field must be completed")
    ).toBeInTheDocument();
    expect(
      await screen.findByText("Full name field must be completed")
    ).toBeInTheDocument();
  });

  it("should throw validation error when invalid email is entered", async () => {
    const { user } = renderWithSetup(
      <UserSettingsForm {...{ ...defaultProps, smtpConfigured: true }} />
    );

    await user.type(
      screen.getByRole("textbox", { name: /email \(required\)/i }),
      "invalid-email"
    );
    await user.click(screen.getByRole("button", { name: "Update" }));

    const emailError = screen.getByText("invalid-email is not a valid email");

    expect(defaultProps.handleSubmit).not.toHaveBeenCalled();
    expect(emailError).toBeInTheDocument();
  });

  it("calls the handleSubmit props with form data", async () => {
    const expectedFormData = {
      email: "email@example.com",
      name: "Jim Example",
    };

    const { user } = renderWithSetup(
      <UserSettingsForm {...{ ...defaultProps, smtpConfigured: true }} />
    );

    await user.type(
      screen.getByRole("textbox", { name: /email \(required\)/i }),
      expectedFormData.email
    );
    await user.type(
      screen.getByRole("textbox", { name: /full name \(required\)/i }),
      expectedFormData.name
    );
    await user.click(screen.getByRole("button", { name: "Update" }));

    expect(defaultProps.handleSubmit).toHaveBeenCalledWith(expectedFormData);
  });

  it("initializes the form with the users data", () => {
    const user = {
      email: "email@example.com",
      name: "Jim Example",
    };
    const props = { ...defaultProps, formData: user };

    render(<UserSettingsForm {...props} />);

    expect(
      screen.getByRole("textbox", { name: /email \(required\)/i })
    ).toHaveValue(user.email);
    expect(
      screen.getByRole("textbox", { name: /full name \(required\)/i })
    ).toHaveValue(user.name);
  });
});
