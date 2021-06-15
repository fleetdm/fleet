import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import ResetPasswordForm from "./ResetPasswordForm";
import { fillInFormInput } from "../../../test/helpers";

describe("ResetPasswordForm - component", () => {
  const newPassword = "p@ssw0rd";

  it("updates component state when the new_password field is changed", () => {
    const form = mount(<ResetPasswordForm handleSubmit={noop} />);

    const newPasswordField = form.find({ name: "new_password" });
    fillInFormInput(newPasswordField, newPassword);

    const { formData } = form.state();
    expect(formData).toMatchObject({ new_password: newPassword });
  });

  it("updates component state when the new_password_confirmation field is changed", () => {
    const form = mount(<ResetPasswordForm handleSubmit={noop} />);

    const newPasswordField = form.find({ name: "new_password_confirmation" });
    fillInFormInput(newPasswordField, newPassword);

    const { formData } = form.state();
    expect(formData).toMatchObject({ new_password_confirmation: newPassword });
  });

  it("it does not submit the form when the form fields have not been filled out", () => {
    const submitSpy = jest.fn();
    const form = mount(<ResetPasswordForm handleSubmit={submitSpy} />);
    const submitBtn = form.find("button");

    submitBtn.simulate("submit");

    const { errors } = form.state();
    expect(errors.new_password).toEqual("New password field must be completed");
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("it does not submit the form when only the new password field has been filled out", () => {
    const submitSpy = jest.fn();
    const form = mount(<ResetPasswordForm handleSubmit={submitSpy} />);
    const newPasswordField = form.find({ name: "new_password" });
    fillInFormInput(newPasswordField, newPassword);
    const submitBtn = form.find("button");

    submitBtn.simulate("submit");

    const { errors } = form.state();
    expect(errors.new_password_confirmation).toEqual(
      "New password confirmation field must be completed"
    );
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("submits the form data when the form is submitted", () => {
    const submitSpy = jest.fn();
    const form = mount(<ResetPasswordForm handleSubmit={submitSpy} />);
    const newPasswordField = form.find({ name: "new_password" });
    const newPasswordConfirmationField = form.find({
      name: "new_password_confirmation",
    });
    const submitBtn = form.find("button");

    fillInFormInput(newPasswordField, newPassword);
    fillInFormInput(newPasswordConfirmationField, newPassword);
    submitBtn.simulate("submit");

    expect(submitSpy).toHaveBeenCalledWith({
      new_password: newPassword,
      new_password_confirmation: newPassword,
    });
  });

  it("does not submit the form if the new password confirmation does not match", () => {
    const submitSpy = jest.fn();
    const form = mount(<ResetPasswordForm handleSubmit={submitSpy} />);
    const newPasswordField = form.find({ name: "new_password" });
    const newPasswordConfirmationField = form.find({
      name: "new_password_confirmation",
    });
    const submitBtn = form.find("button");

    fillInFormInput(newPasswordField, newPassword);
    fillInFormInput(newPasswordConfirmationField, "not my new password");
    submitBtn.simulate("submit");

    expect(submitSpy).not.toHaveBeenCalled();
    expect(form.state().errors).toMatchObject({
      new_password_confirmation: "Passwords do not match",
    });
  });

  it("does not submit the form if the password is invalid", () => {
    const submitSpy = jest.fn();
    const form = mount(<ResetPasswordForm handleSubmit={submitSpy} />);
    const newPasswordField = form.find({ name: "new_password" });
    const newPasswordConfirmationField = form.find({
      name: "new_password_confirmation",
    });
    const submitBtn = form.find("button");
    const invalidPassword = "invalid";

    fillInFormInput(newPasswordField, invalidPassword);
    fillInFormInput(newPasswordConfirmationField, invalidPassword);
    submitBtn.simulate("submit");

    expect(submitSpy).not.toHaveBeenCalled();
    expect(form.state().errors).toMatchObject({
      new_password: "Password must meet the criteria below",
    });
  });
});
