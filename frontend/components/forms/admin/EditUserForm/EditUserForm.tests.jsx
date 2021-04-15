import React from "react";
import { mount } from "enzyme";

import EditUserForm from "./EditUserForm";
import { fillInFormInput } from "../../../../test/helpers";

describe("EditUserForm - form", () => {
  const user = {
    email: "hi@gnar.dog",
    name: "Gnar Dog",
    position: "Head of Everything",
    username: "gnardog",
  };

  it("sends the users changed attributes when the form is submitted", () => {
    const email = "newEmail@gnar.dog";
    const onSubmit = jest.fn();
    const form = mount(
      <EditUserForm formData={user} handleSubmit={onSubmit} />
    ).find("form");
    const emailInput = form.find({ name: "email" });

    fillInFormInput(emailInput, email);
    form.simulate("submit");

    expect(onSubmit).toHaveBeenCalledWith({
      ...user,
      email,
    });
  });
});
