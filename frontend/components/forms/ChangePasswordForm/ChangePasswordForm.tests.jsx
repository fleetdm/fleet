import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import ChangePasswordForm from "components/forms/ChangePasswordForm";

describe("ChangePasswordForm - component", () => {
  const props = {
    handleSubmit: jest.fn(),
    onCancel: jest.fn(),
  };
  it("has the correct fields", () => {
    render(<ChangePasswordForm {...props} />);

    expect(screen.getByLabelText("Original password")).toBeInTheDocument();
    expect(screen.getByLabelText("New password")).toBeInTheDocument();
    expect(
      screen.getByLabelText("New password confirmation")
    ).toBeInTheDocument();

    expect(
      screen.getByRole("button", { name: "Change password" })
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("renders the password fields as HTML password fields", () => {
    render(<ChangePasswordForm {...props} />);

    expect(screen.getByLabelText("Original password")).toHaveAttribute(
      "type",
      "password"
    );
    expect(screen.getByLabelText("New password")).toHaveAttribute(
      "type",
      "password"
    );
    expect(screen.getByLabelText("New password confirmation")).toHaveAttribute(
      "type",
      "password"
    );
  });

  it("should trigger validation for all fields", async () => {
    render(<ChangePasswordForm {...props} />);

    // when
    fireEvent.click(screen.getByRole("button", { name: "Change password" }));
    // then
    expect(props.handleSubmit).not.toHaveBeenCalled();
    expect(
      await screen.findByText("Password must be present")
    ).toBeInTheDocument();
    expect(
      await screen.findByText("New password must be present")
    ).toBeInTheDocument();
    expect(
      await screen.findByText("New password confirmation must be present")
    ).toBeInTheDocument();
  });

  it("does not submit when the new password is invalid", async () => {
    render(<ChangePasswordForm {...props} />);
    const expectedFormData = {
      old_password: "p@ssw0rd",
      new_password: "new_password",
      new_password_confirmation: "new_password",
    };

    // when
    userEvent.type(
      screen.getByLabelText("Original password"),
      expectedFormData.old_password
    );
    userEvent.type(
      screen.getByLabelText("New password"),
      expectedFormData.new_password
    );
    userEvent.type(
      screen.getByLabelText("New password confirmation"),
      expectedFormData.new_password_confirmation
    );
    fireEvent.click(screen.getByRole("button", { name: "Change password" }));
    // then
    expect(props.handleSubmit).not.toHaveBeenCalled();
    expect(
      await screen.findByText("Password must meet the criteria below")
    ).toBeInTheDocument();
  });

  it("does not submit when new confirm password is not matching with new password", async () => {
    render(<ChangePasswordForm {...props} />);
    const expectedFormData = {
      old_password: "p@ssw0rd",
      new_password: "new_password",
      new_password_confirmation: "new_password_1",
    };

    // when
    userEvent.type(
      screen.getByLabelText("Original password"),
      expectedFormData.old_password
    );
    userEvent.type(
      screen.getByLabelText("New password"),
      expectedFormData.new_password
    );
    userEvent.type(
      screen.getByLabelText("New password confirmation"),
      expectedFormData.new_password_confirmation
    );
    fireEvent.click(screen.getByRole("button", { name: "Change password" }));
    // then
    expect(props.handleSubmit).not.toHaveBeenCalled();
    expect(
      await screen.findByText(
        "New password confirmation does not match new password"
      )
    ).toBeInTheDocument();
  });

  it("calls the handleSubmit props with form data", () => {
    render(<ChangePasswordForm {...props} />);

    const expectedFormData = {
      old_password: "p@ssw0rd",
      new_password: "p@ssw0rd1",
      new_password_confirmation: "p@ssw0rd1",
    };

    // when
    userEvent.type(
      screen.getByLabelText("Original password"),
      expectedFormData.old_password
    );
    userEvent.type(
      screen.getByLabelText("New password"),
      expectedFormData.new_password
    );
    userEvent.type(
      screen.getByLabelText("New password confirmation"),
      expectedFormData.new_password_confirmation
    );
    fireEvent.click(screen.getByRole("button", { name: "Change password" }));
    // then
    expect(props.handleSubmit).toHaveBeenCalledWith(expectedFormData);
  });

  it("calls the onCancel prop when CANCEL is clicked", () => {
    render(<ChangePasswordForm {...props} />);

    // when
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    // then
    expect(props.onCancel).toHaveBeenCalled();
  });
});
