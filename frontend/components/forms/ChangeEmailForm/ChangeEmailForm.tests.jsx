import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";
import ChangeEmailForm from "./ChangeEmailForm";

describe("<ChangeEmailForm />", () => {
  const props = {
    handleSubmit: jest.fn(),
    onCancel: jest.fn(),
  };
  const password = "p@ssw0rd";
  it("renders the component correctly", () => {
    render(<ChangeEmailForm {...props} />);

    expect(screen.getByLabelText("Password")).toBeInTheDocument();

    expect(screen.getByRole("button", { name: "Submit" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("it should not submit the form when the password form field have not been filled out", async () => {
    const { user } = renderWithSetup(<ChangeEmailForm {...props} />);

    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(
      await screen.findByText("Password must be present")
    ).toBeInTheDocument();
    expect(props.handleSubmit).not.toHaveBeenCalled();
  });

  it("it should submit the form when the password form field have not been filled out", async () => {
    const { user } = renderWithSetup(<ChangeEmailForm {...props} />);

    await user.type(screen.getByLabelText("Password"), password);
    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(props.handleSubmit).toHaveBeenCalledWith({ password: "p@ssw0rd" });
  });

  it("it should call onCancel function when Cancel button is pressed", async () => {
    const { user } = renderWithSetup(<ChangeEmailForm {...props} />);

    await user.type(screen.getByLabelText("Password"), password);
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(props.onCancel).toHaveBeenCalled();
  });
});
