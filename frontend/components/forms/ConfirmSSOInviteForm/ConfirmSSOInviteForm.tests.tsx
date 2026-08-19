import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import ConfirmSSOInviteForm from "components/forms/ConfirmSSOInviteForm";

describe("ConfirmSSOInviteForm - component", () => {
  const handleSubmitSpy = jest.fn();
  const defaultName = "Test User";
  const email = "test@example.com";

  beforeEach(() => {
    handleSubmitSpy.mockReset();
  });

  it("renders the email field as disabled and prefilled, with a name field", () => {
    render(
      <ConfirmSSOInviteForm
        defaultName={defaultName}
        email={email}
        handleSubmit={handleSubmitSpy}
      />
    );

    const emailInput = screen.getByLabelText("Email") as HTMLInputElement;
    expect(emailInput).toBeInTheDocument();
    expect(emailInput).toBeDisabled();
    expect(emailInput.value).toBe(email);

    const nameInput = screen.getByRole("textbox", {
      name: "Full name",
    }) as HTMLInputElement;
    expect(nameInput).toBeInTheDocument();
    expect(nameInput.value).toBe(defaultName);

    expect(screen.getByRole("button", { name: "Submit" })).toBeInTheDocument();
  });

  it("calls handleSubmit with the name when valid", async () => {
    const { user } = renderWithSetup(
      <ConfirmSSOInviteForm
        defaultName={defaultName}
        email={email}
        handleSubmit={handleSubmitSpy}
      />
    );

    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(handleSubmitSpy).toHaveBeenCalledWith(defaultName);
  });

  it("validates that the name field must be present", async () => {
    const { user } = renderWithSetup(
      <ConfirmSSOInviteForm email={email} handleSubmit={handleSubmitSpy} />
    );

    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(
      await screen.findByText("Full name must be present")
    ).toBeInTheDocument();
    expect(handleSubmitSpy).not.toHaveBeenCalled();
  });

  it("rejects a whitespace-only name and submits a trimmed value otherwise", async () => {
    const { user } = renderWithSetup(
      <ConfirmSSOInviteForm email={email} handleSubmit={handleSubmitSpy} />
    );

    const nameInput = screen.getByRole("textbox", { name: "Full name" });

    await user.type(nameInput, "   ");
    await user.click(screen.getByRole("button", { name: "Submit" }));
    expect(
      await screen.findByText("Full name must be present")
    ).toBeInTheDocument();
    expect(handleSubmitSpy).not.toHaveBeenCalled();

    await user.type(nameInput, "Padded Name  ");
    await user.click(screen.getByRole("button", { name: "Submit" }));
    expect(handleSubmitSpy).toHaveBeenCalledWith("Padded Name");
  });
});
