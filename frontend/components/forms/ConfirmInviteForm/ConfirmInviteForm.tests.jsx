import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import ConfirmInviteForm from "components/forms/ConfirmInviteForm";
import userEvent from "@testing-library/user-event";

describe("ConfirmInviteForm - component", () => {
  const handleSubmitSpy = jest.fn();
  const inviteToken = "abc123";
  const formData = { invite_token: inviteToken };

  it("renders", () => {
    render(
      <ConfirmInviteForm formData={formData} handleSubmit={handleSubmitSpy} />
    );
    expect(
      screen.getByRole("textbox", { name: "Full name" })
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.getByLabelText("Confirm password")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Submit" })).toBeInTheDocument();
  });

  it("renders the base error", () => {
    const baseError = "Unable to authenticate the current user";
    render(
      <ConfirmInviteForm
        serverErrors={{ base: baseError }}
        handleSubmit={handleSubmitSpy}
      />
    );

    expect(screen.getByText(baseError)).toBeInTheDocument();
  });

  it("calls the handleSubmit prop with the invite_token when valid", () => {
    render(
      <ConfirmInviteForm formData={formData} handleSubmit={handleSubmitSpy} />
    );
    // when
    userEvent.type(
      screen.getByRole("textbox", { name: "Full name" }),
      "Gnar Dog"
    );
    userEvent.type(screen.getByLabelText("Password"), "p@ssw0rd");
    userEvent.type(screen.getByLabelText("Confirm password"), "p@ssw0rd");
    fireEvent.click(screen.getByRole("button", { name: "Submit" }));
    // then
    expect(handleSubmitSpy).toHaveBeenCalledWith({
      ...formData,
      name: "Gnar Dog",
      password: "p@ssw0rd",
      password_confirmation: "p@ssw0rd",
    });
  });

  describe("name input", () => {
    it("validates the field must be present", async () => {
      render(
        <ConfirmInviteForm formData={formData} handleSubmit={handleSubmitSpy} />
      );
      // when
      userEvent.type(screen.getByRole("textbox", { name: "Full name" }), "");
      fireEvent.click(screen.getByRole("button", { name: "Submit" }));
      // then
      expect(
        await screen.findByText("Full name must be present")
      ).toBeInTheDocument();
    });
  });

  describe("password input", () => {
    it("validates the field must be present", async () => {
      render(
        <ConfirmInviteForm formData={formData} handleSubmit={handleSubmitSpy} />
      );
      // when
      userEvent.type(screen.getByLabelText("Password"), "");
      fireEvent.click(screen.getByRole("button", { name: "Submit" }));
      // then
      expect(
        await screen.findByText("Password must be present")
      ).toBeInTheDocument();
    });
  });

  describe("password_confirmation input", () => {
    it("validates the password_confirmation matches the password", async () => {
      render(
        <ConfirmInviteForm formData={formData} handleSubmit={handleSubmitSpy} />
      );

      // when
      userEvent.type(screen.getByLabelText("Password"), "p@ssw0rd");
      userEvent.type(
        screen.getByLabelText("Confirm password"),
        "another password"
      );
      fireEvent.click(screen.getByRole("button", { name: "Submit" }));
      // then
      expect(
        await screen.findByText("Password confirmation does not match password")
      ).toBeInTheDocument();
    });

    it("validates the field must be present", async () => {
      render(
        <ConfirmInviteForm formData={formData} handleSubmit={handleSubmitSpy} />
      );
      // when
      userEvent.type(screen.getByLabelText("Confirm password"), "");
      fireEvent.click(screen.getByRole("button", { name: "Submit" }));
      // then
      expect(
        await screen.findByText("Password confirmation must be present")
      ).toBeInTheDocument();
    });
  });
});
