import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import AdminDetails from "components/forms/RegistrationForm/AdminDetails";

describe("AdminDetails - form", () => {
  const onSubmitSpy = jest.fn();
  it("renders", () => {
    render(<AdminDetails handleSubmit={onSubmitSpy} />);

    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.getByLabelText("Confirm password")).toBeInTheDocument();
    expect(
      screen.getByRole("textbox", { name: "Full name" })
    ).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "Email" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next" })).toBeInTheDocument();
  });

  it("validates missing fields", async () => {
    const { user } = renderWithSetup(
      <AdminDetails handleSubmit={onSubmitSpy} currentPage />
    );

    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(onSubmitSpy).not.toHaveBeenCalled();
    expect(screen.getByText("Email must be present")).toBeInTheDocument();
    expect(screen.getByText("Password must be present")).toBeInTheDocument();
    expect(
      screen.getByText("Password confirmation must be present")
    ).toBeInTheDocument();
    expect(screen.getByText("Full name must be present")).toBeInTheDocument();
  });

  it("validates the email field", async () => {
    const { user } = renderWithSetup(
      <AdminDetails handleSubmit={onSubmitSpy} currentPage />
    );

    await user.type(
      screen.getByRole("textbox", { name: "Email" }),
      "invalid-email"
    );
    await user.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(onSubmitSpy).not.toHaveBeenCalled();
    expect(screen.getByText("Email must be a valid email")).toBeInTheDocument();
  });

  it("validates the password fields match", async () => {
    const { user } = renderWithSetup(
      <AdminDetails handleSubmit={onSubmitSpy} currentPage />
    );

    await user.type(screen.getByLabelText("Password"), "p@ssw0rd");
    await user.type(screen.getByLabelText("Confirm password"), "password123");
    await user.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(onSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Password confirmation does not match password")
    ).toBeInTheDocument();
  });

  it("validates the password is not too long", async () => {
    const { user } = renderWithSetup(
      <AdminDetails handleSubmit={onSubmitSpy} currentPage />
    );

    await user.type(
      screen.getByLabelText("Password"),
      "asasasasasasasasasasasasasasasasasasasasasasasas1!"
    );
    await user.type(
      screen.getByLabelText("Confirm password"),
      "asasasasasasasasasasasasasasasasasasasasasasasas1!"
    );
    await user.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(onSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Password is over the character limit")
    ).toBeInTheDocument();
  });

  it("validates the password field", async () => {
    const { user } = renderWithSetup(
      <AdminDetails handleSubmit={onSubmitSpy} currentPage />
    );

    await user.type(screen.getByLabelText("Password"), "invalidpassw0rd");
    await user.type(
      screen.getByLabelText("Confirm password"),
      "invalidpassw0rd"
    );
    await user.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(onSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Password must meet the criteria below")
    ).toBeInTheDocument();
  });

  it("submits the form when valid", async () => {
    const { user } = renderWithSetup(
      <AdminDetails handleSubmit={onSubmitSpy} currentPage />
    );

    await user.type(
      screen.getByRole("textbox", { name: "Email" }),
      "hi@gnar.dog"
    );
    await user.type(screen.getByLabelText("Password"), "password123#");
    await user.type(screen.getByLabelText("Confirm password"), "password123#");
    await user.type(
      screen.getByRole("textbox", { name: "Full name" }),
      "Gnar Dog"
    );
    await user.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(onSubmitSpy).toHaveBeenCalledWith({
      email: "hi@gnar.dog",
      name: "Gnar Dog",
      password: "password123#",
      password_confirmation: "password123#",
    });
  });
});
