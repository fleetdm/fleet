import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import ChangePasswordForm from "components/forms/ChangePasswordForm";
import helpers from "test/helpers";

const { fillInFormInput, itBehavesLikeAFormInputElement } = helpers;

describe("ChangePasswordForm - component", () => {
  it("has the correct fields", () => {
    const form = mount(
      <ChangePasswordForm handleSubmit={noop} onCancel={noop} />
    );

    itBehavesLikeAFormInputElement(form, "old_password");
    itBehavesLikeAFormInputElement(form, "new_password");
    itBehavesLikeAFormInputElement(form, "new_password_confirmation");
  });

  it("renders the password fields as HTML password fields", () => {
    const form = mount(
      <ChangePasswordForm handleSubmit={noop} onCancel={noop} />
    );
    const passwordField = form.find('input[name="old_password"]');
    const newPasswordField = form.find('input[name="new_password"]');
    const newPasswordConfirmationField = form.find(
      'input[name="new_password_confirmation"]'
    );

    expect(passwordField.prop("type")).toEqual("password");
    expect(newPasswordField.prop("type")).toEqual("password");
    expect(newPasswordConfirmationField.prop("type")).toEqual("password");
  });

  it("calls the handleSubmit props with form data", () => {
    const handleSubmitSpy = jest.fn();
    const form = mount(
      <ChangePasswordForm handleSubmit={handleSubmitSpy} onCancel={noop} />
    ).find("form");
    const expectedFormData = {
      old_password: "p@ssw0rd",
      new_password: "p@ssw0rd1",
      new_password_confirmation: "p@ssw0rd1",
    };
    const passwordInput = form.find({ name: "old_password" }).find("input");
    const newPasswordInput = form.find({ name: "new_password" }).find("input");
    const newPasswordConfirmationInput = form
      .find({ name: "new_password_confirmation" })
      .find("input");

    fillInFormInput(passwordInput, expectedFormData.old_password);
    fillInFormInput(newPasswordInput, expectedFormData.new_password);
    fillInFormInput(
      newPasswordConfirmationInput,
      expectedFormData.new_password_confirmation
    );

    form.simulate("submit");

    expect(handleSubmitSpy).toHaveBeenCalledWith(expectedFormData);
  });

  it("calls the onCancel prop when CANCEL is clicked", () => {
    const onCancelSpy = jest.fn();
    const form = mount(
      <ChangePasswordForm handleSubmit={noop} onCancel={onCancelSpy} />
    ).find("form");
    const cancelBtn = form
      .find("Button")
      .findWhere((n) => n.prop("children") === "Cancel")
      .find("button");

    cancelBtn.simulate("click");

    expect(onCancelSpy).toHaveBeenCalled();
  });

  it("does not submit when the new password is invalid", () => {
    const handleSubmitSpy = jest.fn();
    const component = mount(
      <ChangePasswordForm handleSubmit={handleSubmitSpy} onCancel={noop} />
    );
    const form = component.find("form");
    const expectedFormData = {
      old_password: "p@ssw0rd",
      new_password: "new_password",
      new_password_confirmation: "new_password",
    };
    const passwordInput = form.find({ name: "old_password" }).find("input");
    const newPasswordInput = form.find({ name: "new_password" }).find("input");
    const newPasswordConfirmationInput = form
      .find({ name: "new_password_confirmation" })
      .find("input");

    fillInFormInput(passwordInput, expectedFormData.old_password);
    fillInFormInput(newPasswordInput, expectedFormData.new_password);
    fillInFormInput(
      newPasswordConfirmationInput,
      expectedFormData.new_password_confirmation
    );

    form.simulate("submit");

    expect(handleSubmitSpy).not.toHaveBeenCalled();

    expect(component.state("errors")).toMatchObject({
      new_password: "Password must meet the criteria below",
    });
  });
});
