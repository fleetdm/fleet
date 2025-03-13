import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import ConfirmInviteForm from "components/forms/ConfirmInviteForm";

describe("ConfirmInviteForm - component", () => {
  const handleSubmitSpy = jest.fn();
  const defaultFormData = { name: "Test User" };

  it("renders", () => {
    render(
      <ConfirmInviteForm
        defaultFormData={defaultFormData}
        handleSubmit={handleSubmitSpy}
      />
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
        ancestorError={baseError}
        handleSubmit={handleSubmitSpy}
      />
    );

    expect(screen.getByText(baseError)).toBeInTheDocument();
  });

  it("calls the handleSubmit prop when valid", async () => {
    const { user } = renderWithSetup(
      <ConfirmInviteForm
        defaultFormData={defaultFormData}
        handleSubmit={handleSubmitSpy}
      />
    );

    await user.type(screen.getByLabelText("Password"), "p@ssw0rd");
    await user.type(screen.getByLabelText("Confirm password"), "p@ssw0rd");
    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(handleSubmitSpy).toHaveBeenCalledWith({
      ...defaultFormData,
      password: "p@ssw0rd",
      password_confirmation: "p@ssw0rd",
    });
  });

  describe("name input", () => {
    it("validates the field must be present", async () => {
      const { user } = renderWithSetup(
        <ConfirmInviteForm
          defaultFormData={{ ...defaultFormData, ...{ name: "" } }}
          handleSubmit={handleSubmitSpy}
        />
      );

      await user.click(screen.getByRole("button", { name: "Submit" }));

      expect(
        await screen.findByText("Full name must be present")
      ).toBeInTheDocument();
    });
  });

  describe("password input", () => {
    it("validates the field must be present", async () => {
      const { user } = renderWithSetup(
        <ConfirmInviteForm
          defaultFormData={defaultFormData}
          handleSubmit={handleSubmitSpy}
        />
      );

      await user.click(screen.getByRole("button", { name: "Submit" }));

      expect(
        await screen.findByText("Password must be present")
      ).toBeInTheDocument();
    });
  });

  describe("password_confirmation input", () => {
    it("validates the password_confirmation matches the password", async () => {
      const { user } = renderWithSetup(
        <ConfirmInviteForm
          defaultFormData={defaultFormData}
          handleSubmit={handleSubmitSpy}
        />
      );

      await user.type(screen.getByLabelText("Password"), "p@ssw0rd");
      await user.type(
        screen.getByLabelText("Confirm password"),
        "another password"
      );
      await user.click(screen.getByRole("button", { name: "Submit" }));

      const passwordError = screen.getByText(
        "Password confirmation does not match password"
      );
      expect(passwordError).toBeInTheDocument();
    });

    it("validates the field must be present", async () => {
      const { user } = renderWithSetup(
        <ConfirmInviteForm
          defaultFormData={defaultFormData}
          handleSubmit={handleSubmitSpy}
        />
      );

      await user.click(screen.getByRole("button", { name: "Submit" }));

      const passwordError = screen.getByText(
        "Password confirmation must be present"
      );

      expect(passwordError).toBeInTheDocument();
    });
  });
});
