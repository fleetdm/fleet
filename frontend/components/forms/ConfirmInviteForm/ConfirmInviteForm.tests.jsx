import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import ConfirmInviteForm from "components/forms/ConfirmInviteForm";
import { fillInFormInput } from "test/helpers";

describe("ConfirmInviteForm - component", () => {
  const handleSubmitSpy = jest.fn();
  const inviteToken = "abc123";
  const formData = { invite_token: inviteToken };
  const form = mount(
    <ConfirmInviteForm formData={formData} handleSubmit={handleSubmitSpy} />
  );

  const nameInput = form.find({ name: "name" }).find("input");
  const passwordConfirmationInput = form
    .find({ name: "password_confirmation" })
    .find("input");
  const passwordInput = form.find({ name: "password" }).find("input");
  const submitBtn = form.find("button");

  it("renders", () => {
    expect(form.length).toEqual(1);
  });

  it("renders the base error", () => {
    const baseError = "Unable to authenticate the current user";
    const formWithError = mount(
      <ConfirmInviteForm
        serverErrors={{ base: baseError }}
        handleSubmit={noop}
      />
    );
    const formWithoutError = mount(<ConfirmInviteForm handleSubmit={noop} />);

    expect(formWithError.text()).toContain(baseError);
    expect(formWithoutError.text()).not.toContain(baseError);
  });

  it("calls the handleSubmit prop with the invite_token when valid", () => {
    fillInFormInput(nameInput, "Gnar Dog");
    fillInFormInput(passwordInput, "p@ssw0rd");
    fillInFormInput(passwordConfirmationInput, "p@ssw0rd");
    submitBtn.simulate("click");

    expect(handleSubmitSpy).toHaveBeenCalledWith({
      ...formData,
      name: "Gnar Dog",
      password: "p@ssw0rd",
      password_confirmation: "p@ssw0rd",
    });
  });

  describe("name input", () => {
    it("changes form state on change", () => {
      fillInFormInput(nameInput, "Gnar Dog");

      expect(form.state().formData).toMatchObject({ name: "Gnar Dog" });
    });

    it("validates the field must be present", () => {
      fillInFormInput(nameInput, "");
      form.find("button").simulate("click");

      expect(form.state().errors).toMatchObject({
        name: "Full name must be present",
      });
    });
  });

  describe("password input", () => {
    it("changes form state on change", () => {
      fillInFormInput(passwordInput, "p@ssw0rd");

      expect(form.state().formData).toMatchObject({ password: "p@ssw0rd" });
    });

    it("validates the field must be present", () => {
      fillInFormInput(passwordInput, "");
      form.find("button").simulate("click");

      expect(form.state().errors).toMatchObject({
        password: "Password must be present",
      });
    });
  });

  describe("password_confirmation input", () => {
    it("changes form state on change", () => {
      fillInFormInput(passwordConfirmationInput, "p@ssw0rd");

      expect(form.state().formData).toMatchObject({
        password_confirmation: "p@ssw0rd",
      });
    });

    it("validates the password_confirmation matches the password", () => {
      fillInFormInput(passwordInput, "p@ssw0rd");
      fillInFormInput(passwordConfirmationInput, "another-password");
      form.find("button").simulate("click");

      expect(form.state().errors).toMatchObject({
        password_confirmation: "Password confirmation does not match password",
      });
    });

    it("validates the field must be present", () => {
      fillInFormInput(passwordConfirmationInput, "");
      form.find("button").simulate("click");

      expect(form.state().errors).toMatchObject({
        password_confirmation: "Password confirmation must be present",
      });
    });
  });
});
