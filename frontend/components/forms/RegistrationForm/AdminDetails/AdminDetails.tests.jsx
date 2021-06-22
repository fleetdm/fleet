import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import AdminDetails from "components/forms/RegistrationForm/AdminDetails";
import { fillInFormInput, itBehavesLikeAFormInputElement } from "test/helpers";

describe("AdminDetails - form", () => {
  let form = mount(<AdminDetails handleSubmit={noop} />);

  describe("full name input", () => {
    it("renders an input field", () => {
      itBehavesLikeAFormInputElement(form, "name");
    });
  });

  describe("password input", () => {
    it("renders an input field", () => {
      itBehavesLikeAFormInputElement(form, "password");
    });
  });

  describe("password confirmation input", () => {
    it("renders an input field", () => {
      itBehavesLikeAFormInputElement(form, "password_confirmation");
    });
  });

  describe("email input", () => {
    it("renders an input field", () => {
      itBehavesLikeAFormInputElement(form, "email");
    });
  });

  describe("submitting the form", () => {
    it("validates missing fields", () => {
      const onSubmitSpy = jest.fn();
      form = mount(<AdminDetails handleSubmit={onSubmitSpy} />);
      const htmlForm = form.find("form");

      htmlForm.simulate("submit");

      expect(onSubmitSpy).not.toHaveBeenCalled();
      expect(form.state().errors).toMatchObject({
        email: "Email must be present",
        password: "Password must be present",
        password_confirmation: "Password confirmation must be present",
        name: "Full name must be present",
      });
    });

    it("validates the email field", () => {
      const onSubmitSpy = jest.fn();
      form = mount(<AdminDetails handleSubmit={onSubmitSpy} />);
      const emailField = form.find({ name: "email" }).find("input");
      const htmlForm = form.find("form");

      fillInFormInput(emailField, "invalid-email");
      htmlForm.simulate("submit");

      expect(onSubmitSpy).not.toHaveBeenCalled();
      expect(form.state().errors).toMatchObject({
        email: "Email must be a valid email",
      });
    });

    it("validates the password fields match", () => {
      const onSubmitSpy = jest.fn();
      form = mount(<AdminDetails handleSubmit={onSubmitSpy} />);
      const passwordConfirmationField = form
        .find({ name: "password_confirmation" })
        .find("input");
      const passwordField = form.find({ name: "password" }).find("input");
      const htmlForm = form.find("form");

      fillInFormInput(passwordField, "p@ssw0rd");
      fillInFormInput(passwordConfirmationField, "password123");
      htmlForm.simulate("submit");

      expect(onSubmitSpy).not.toHaveBeenCalled();
      expect(form.state().errors).toMatchObject({
        password_confirmation: "Password confirmation does not match password",
      });
    });

    it("validates the password field", () => {
      const onSubmitSpy = jest.fn();
      form = mount(<AdminDetails handleSubmit={onSubmitSpy} />);
      const passwordConfirmationField = form
        .find({ name: "password_confirmation" })
        .find("input");
      const passwordField = form.find({ name: "password" }).find("input");
      const htmlForm = form.find("form");

      fillInFormInput(passwordField, "passw0rd");
      fillInFormInput(passwordConfirmationField, "passw0rd");
      htmlForm.simulate("submit");

      expect(onSubmitSpy).not.toHaveBeenCalled();
      expect(form.state().errors).toMatchObject({
        password: "Password must meet the criteria below",
      });
    });

    it("submits the form when valid", () => {
      const onSubmitSpy = jest.fn();
      form = mount(<AdminDetails handleSubmit={onSubmitSpy} />);
      const emailField = form.find({ name: "email" }).find("input");
      const passwordConfirmationField = form
        .find({ name: "password_confirmation" })
        .find("input");
      const passwordField = form.find({ name: "password" }).find("input");
      const nameField = form.find({ name: "name" }).find("input");
      const htmlForm = form.find("form");

      fillInFormInput(emailField, "hi@gnar.dog");
      fillInFormInput(passwordField, "p@ssw0rd");
      fillInFormInput(passwordConfirmationField, "p@ssw0rd");
      fillInFormInput(nameField, "Gnar Dog");
      htmlForm.simulate("submit");

      expect(onSubmitSpy).toHaveBeenCalled();
    });
  });
});
