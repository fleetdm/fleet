import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";
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
    const { user } = renderWithSetup(<ChangePasswordForm {...props} />);
    const expectedFormData = {
      old_password: "p@ssw0rd",
      new_password: "new_password",
      new_password_confirmation: "new_password",
    };

    await user.type(
      screen.getByLabelText("Original password"),
      expectedFormData.old_password
    );
    await user.type(
      screen.getByLabelText("New password"),
      expectedFormData.new_password
    );
    await user.type(
      screen.getByLabelText("New password confirmation"),
      expectedFormData.new_password_confirmation
    );
    await user.click(screen.getByRole("button", { name: "Change password" }));

    const passwordError = screen.getByText(
      "Password must meet the criteria below"
    );
    expect(props.handleSubmit).not.toHaveBeenCalled();
    expect(passwordError).toBeInTheDocument();
  });

  it("does not submit when new confirm password is not matching with new password", async () => {
    const { user } = renderWithSetup(<ChangePasswordForm {...props} />);
    const expectedFormData = {
      old_password: "p@ssw0rd",
      new_password: "new_password",
      new_password_confirmation: "new_password_1",
    };

    await user.type(
      screen.getByLabelText("Original password"),
      expectedFormData.old_password
    );
    await user.type(
      screen.getByLabelText("New password"),
      expectedFormData.new_password
    );
    await user.type(
      screen.getByLabelText("New password confirmation"),
      expectedFormData.new_password_confirmation
    );
    await user.click(screen.getByRole("button", { name: "Change password" }));

    const passwordConfirmError = screen.getByText(
      "New password confirmation does not match new password"
    );
    expect(props.handleSubmit).not.toHaveBeenCalled();
    expect(passwordConfirmError).toBeInTheDocument();
  });

  it("calls the handleSubmit props with form data", async () => {
    const { user } = renderWithSetup(<ChangePasswordForm {...props} />);

    const expectedFormData = {
      old_password: "password123#",
      new_password: "password123!",
      new_password_confirmation: "password123!",
    };

    await user.type(
      screen.getByLabelText("Original password"),
      expectedFormData.old_password
    );
    await user.type(
      screen.getByLabelText("New password"),
      expectedFormData.new_password
    );
    await user.type(
      screen.getByLabelText("New password confirmation"),
      expectedFormData.new_password_confirmation
    );
    await user.click(screen.getByRole("button", { name: "Change password" }));
    // then
    expect(props.handleSubmit).toHaveBeenCalledWith(expectedFormData);
  });

  it("calls the onCancel prop when CANCEL is clicked", async () => {
    const { user } = renderWithSetup(<ChangePasswordForm {...props} />);

    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(props.onCancel).toHaveBeenCalled();
  });
});
