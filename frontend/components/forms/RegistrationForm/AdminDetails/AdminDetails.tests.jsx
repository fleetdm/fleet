import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import AdminDetails from "components/forms/RegistrationForm/AdminDetails";

describe("AdminDetails - form", () => {
  const onSubmitSpy = jest.fn();
  it("renders", () => {
    render(<AdminDetails handleSubmit={onSubmitSpy} />);

    expect(screen.getByPlaceholderText("Password")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Confirm password")).toBeInTheDocument();
    expect(
      screen.getByRole("textbox", { name: "Full name" })
    ).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "Email" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next" })).toBeInTheDocument();
  });

  it("validates missing fields", () => {
    render(<AdminDetails handleSubmit={onSubmitSpy} currentPage />);

    // when
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(onSubmitSpy).not.toHaveBeenCalled();
    expect(screen.getByText("Email must be present")).toBeInTheDocument();
    expect(screen.getByText("Password must be present")).toBeInTheDocument();
    expect(
      screen.getByText("Password confirmation must be present")
    ).toBeInTheDocument();
    expect(screen.getByText("Full name must be present")).toBeInTheDocument();
  });

  it("validates the email field", () => {
    render(<AdminDetails handleSubmit={onSubmitSpy} currentPage />);

    // when
    userEvent.type(
      screen.getByRole("textbox", { name: "Email" }),
      "invalid-email"
    );
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(onSubmitSpy).not.toHaveBeenCalled();
    expect(screen.getByText("Email must be a valid email")).toBeInTheDocument();
  });

  it("validates the password fields match", () => {
    render(<AdminDetails handleSubmit={onSubmitSpy} currentPage />);
    // when
    userEvent.type(screen.getByPlaceholderText("Password"), "p@ssw0rd");
    userEvent.type(
      screen.getByPlaceholderText("Confirm password"),
      "password123"
    );
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(onSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Password confirmation does not match password")
    ).toBeInTheDocument();
  });

  it("validates the password field", () => {
    render(<AdminDetails handleSubmit={onSubmitSpy} currentPage />);
    // when
    userEvent.type(screen.getByPlaceholderText("Password"), "passw0rd");
    userEvent.type(screen.getByPlaceholderText("Confirm password"), "passw0rd");
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(onSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Password must meet the criteria below")
    ).toBeInTheDocument();
  });

  it("submits the form when valid", () => {
    render(<AdminDetails handleSubmit={onSubmitSpy} currentPage />);
    // when
    userEvent.type(
      screen.getByRole("textbox", { name: "Email" }),
      "hi@gnar.dog"
    );
    userEvent.type(screen.getByPlaceholderText("Password"), "p@ssw0rd");
    userEvent.type(screen.getByPlaceholderText("Confirm password"), "p@ssw0rd");
    userEvent.type(
      screen.getByRole("textbox", { name: "Full name" }),
      "Gnar Dog"
    );
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(onSubmitSpy).toHaveBeenCalledWith({
      email: "hi@gnar.dog",
      name: "Gnar Dog",
      password: "p@ssw0rd",
      password_confirmation: "p@ssw0rd",
    });
  });
});
