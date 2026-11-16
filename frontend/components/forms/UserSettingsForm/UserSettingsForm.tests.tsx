import { render, screen } from "@testing-library/react";
import React from "react";

import { renderWithSetup } from "test/test-utils";

import UserSettingsForm from "./UserSettingsForm";

describe("UserSettingsForm - component", () => {
  const defaultProps = {
    handleSubmit: jest.fn(),
    onCancel: jest.fn(),
    smtpConfigured: true,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders fields and buttons", () => {
    render(<UserSettingsForm {...defaultProps} />);

    expect(screen.getByRole("textbox", { name: /email/i })).toBeInTheDocument();
    expect(
      screen.getByRole("textbox", { name: /full name/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("textbox", { name: /position/i })
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Update" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("surfaces required-field errors on submit", async () => {
    const { user } = renderWithSetup(<UserSettingsForm {...defaultProps} />);

    await user.click(screen.getByRole("button", { name: "Update" }));

    expect(await screen.findByText("Enter your email")).toBeInTheDocument();
    expect(screen.getByText("Enter your name")).toBeInTheDocument();
    expect(defaultProps.handleSubmit).not.toHaveBeenCalled();
  });

  it("validates an invalid email on blur", async () => {
    const { user } = renderWithSetup(<UserSettingsForm {...defaultProps} />);

    await user.type(
      screen.getByRole("textbox", { name: /email/i }),
      "invalid-email"
    );
    await user.tab();

    expect(await screen.findByText("Enter a valid email")).toBeInTheDocument();
  });

  it("clears an error when the field is focused again", async () => {
    const { container, user } = renderWithSetup(
      <UserSettingsForm {...defaultProps} />
    );

    await user.click(screen.getByRole("button", { name: "Update" }));
    expect(await screen.findByText("Enter your name")).toBeInTheDocument();

    const nameInput = container.querySelector<HTMLInputElement>(
      'input[name="name"]'
    );
    expect(nameInput).not.toBeNull();
    await user.click(nameInput!);

    expect(screen.queryByText("Enter your name")).not.toBeInTheDocument();
  });

  it("calls handleSubmit with trimmed form data", async () => {
    const { user } = renderWithSetup(<UserSettingsForm {...defaultProps} />);

    await user.type(
      screen.getByRole("textbox", { name: /email/i }),
      "  email@example.com  "
    );
    await user.type(
      screen.getByRole("textbox", { name: /full name/i }),
      "Jim Example"
    );
    await user.click(screen.getByRole("button", { name: "Update" }));

    expect(defaultProps.handleSubmit).toHaveBeenCalledWith({
      email: "email@example.com",
      name: "Jim Example",
      position: undefined,
    });
  });

  it("initializes the form with the user's data", () => {
    const defaultData = {
      email: "email@example.com",
      name: "Jim Example",
      position: "Engineer",
    };

    render(<UserSettingsForm {...defaultProps} defaultData={defaultData} />);

    expect(screen.getByRole("textbox", { name: /email/i })).toHaveValue(
      defaultData.email
    );
    expect(screen.getByRole("textbox", { name: /full name/i })).toHaveValue(
      defaultData.name
    );
    expect(screen.getByRole("textbox", { name: /position/i })).toHaveValue(
      defaultData.position
    );
  });

  it("skips email validation when SMTP is not configured", async () => {
    const { user } = renderWithSetup(
      <UserSettingsForm
        {...defaultProps}
        smtpConfigured={false}
        defaultData={{ email: "existing@example.com", name: "Jim" }}
      />
    );

    await user.click(screen.getByRole("button", { name: "Update" }));

    expect(screen.queryByText("Enter your email")).not.toBeInTheDocument();
    expect(defaultProps.handleSubmit).toHaveBeenCalled();
  });
});
