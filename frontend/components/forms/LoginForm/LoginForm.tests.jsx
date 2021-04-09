import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";
import LoginForm from "./LoginForm";
import { fillInFormInput } from "../../../test/helpers";

describe("LoginForm - component", () => {
  const settings = { sso_enabled: false };

  it("renders the base error", () => {
    const baseError = "Unable to authenticate the current user";
    const formWithError = mount(
      <LoginForm
        serverErrors={{ base: baseError }}
        handleSubmit={noop}
        ssoSettings={settings}
      />
    );
    const formWithoutError = mount(
      <LoginForm handleSubmit={noop} ssoSettings={settings} />
    );

    expect(formWithError.text()).toContain(baseError);
    expect(formWithoutError.text()).not.toContain(baseError);
  });

  it("renders 2 InputField components", () => {
    const form = mount(
      <LoginForm handleSubmit={noop} ssoSettings={settings} />
    );

    expect(form.find("InputFieldWithIcon").length).toEqual(2);
  });

  it("updates component state when the username field is changed", () => {
    const form = mount(
      <LoginForm handleSubmit={noop} ssoSettings={settings} />
    );
    const username = "hi@thegnar.co";

    const usernameField = form.find({ name: "username" });
    fillInFormInput(usernameField, username);

    const { formData } = form.state();
    expect(formData).toMatchObject({ username });
  });

  it("updates component state when the password field is changed", () => {
    const form = mount(
      <LoginForm handleSubmit={noop} ssoSettings={settings} />
    );

    const passwordField = form.find({ name: "password" });
    fillInFormInput(passwordField, "hello");

    const { formData } = form.state();
    expect(formData).toMatchObject({
      password: "hello",
    });
  });

  it("it does not submit the form when the form fields have not been filled out", () => {
    const submitSpy = jest.fn();
    const form = mount(
      <LoginForm handleSubmit={submitSpy} ssoSettings={settings} />
    );
    const submitBtn = form.find("button");

    submitBtn.simulate("click");

    expect(form.state().errors).toMatchObject({
      username: "Username or email field must be completed",
    });
    expect(submitSpy).not.toHaveBeenCalled();
  });

  it("submits the form data when form is submitted", () => {
    const submitSpy = jest.fn();

    const form = mount(
      <LoginForm handleSubmit={submitSpy} ssoSettings={settings} />
    );
    const usernameField = form.find({ name: "username" });
    const passwordField = form.find({ name: "password" });
    const submitBtn = form.find("button");

    fillInFormInput(usernameField, "my@email.com");
    fillInFormInput(passwordField, "p@ssw0rd");
    submitBtn.simulate("submit");

    expect(submitSpy).toHaveBeenCalledWith({
      username: "my@email.com",
      password: "p@ssw0rd",
    });
  });
});
