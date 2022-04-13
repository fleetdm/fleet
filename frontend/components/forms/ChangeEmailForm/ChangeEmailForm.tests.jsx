import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

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
    render(<ChangeEmailForm {...props} />);

    // when
    fireEvent.click(screen.getByRole("button", { name: "Submit" }));
    // then
    expect(
      await screen.findByText("Password must be present")
    ).toBeInTheDocument();
    expect(props.handleSubmit).not.toHaveBeenCalled();
  });

  it("it should submit the form when the password form field have not been filled out", async () => {
    render(<ChangeEmailForm {...props} />);

    // when
    userEvent.type(screen.getByLabelText("Password"), password);
    fireEvent.click(screen.getByRole("button", { name: "Submit" }));
    // then
    expect(props.handleSubmit).toHaveBeenCalledWith({ password: "p@ssw0rd" });
  });

  it("it should call onCancel function when Cancel button is pressed", async () => {
    render(<ChangeEmailForm {...props} />);

    // when
    userEvent.type(screen.getByLabelText("Password"), password);
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    // then
    expect(props.onCancel).toHaveBeenCalled();
  });
});
