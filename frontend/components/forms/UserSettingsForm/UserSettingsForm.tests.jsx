import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import UserSettingsForm from "components/forms/UserSettingsForm";
import helpers from "test/helpers";

const { fillInFormInput, itBehavesLikeAFormInputElement } = helpers;

describe("UserSettingsForm - component", () => {
  const defaultProps = {
    handleSubmit: noop,
    onCancel: noop,
  };

  it("has the correct fields", () => {
    const form = mount(<UserSettingsForm {...defaultProps} />);

    itBehavesLikeAFormInputElement(form, "email");
    itBehavesLikeAFormInputElement(form, "name");
  });

  it("calls the handleSubmit props with form data", () => {
    const handleSubmitSpy = jest.fn();
    const props = { ...defaultProps, handleSubmit: handleSubmitSpy };
    const form = mount(<UserSettingsForm {...props} />);
    const expectedFormData = {
      email: "email@example.com",
      name: "Jim Example",
    };
    const emailInput = form.find({ name: "email" }).find("input");
    const nameInput = form.find({ name: "name" }).find("input");

    fillInFormInput(emailInput, expectedFormData.email);
    fillInFormInput(nameInput, expectedFormData.name);

    form.find("form").simulate("submit");

    expect(handleSubmitSpy).toHaveBeenCalledWith(expectedFormData);
  });

  it("initializes the form with the users data", () => {
    const user = {
      email: "email@example.com",
      name: "Jim Example",
    };
    const props = { ...defaultProps, formData: user };
    const form = mount(<UserSettingsForm {...props} />);

    expect(form.state().formData).toEqual(user);
  });
});
