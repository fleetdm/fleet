import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import { fillInFormInput } from "test/helpers";
import targetMock from "test/target_mock";
import PackForm from "./index";

describe("PackForm - component", () => {
  beforeEach(targetMock);

  it("renders the base error", () => {
    const baseError = "Pack already exists";
    const formWithError = mount(
      <PackForm serverErrors={{ base: baseError }} handleSubmit={noop} />
    );
    const formWithoutError = mount(<PackForm handleSubmit={noop} />);

    expect(formWithError.text()).toContain(baseError);
    expect(formWithoutError.text()).not.toContain(baseError);
  });

  it("renders the correct components", () => {
    const form = mount(<PackForm />);

    expect(form.find("InputField").length).toEqual(2);
    expect(form.find("SelectTargetsDropdown").length).toEqual(1);
    expect(form.find("Button").length).toEqual(1);
  });

  it("validates the query pack name field", () => {
    const handleSubmitSpy = jest.fn();
    const form = mount(<PackForm handleSubmit={handleSubmitSpy} />);

    form.find("form").simulate("submit");

    expect(handleSubmitSpy).not.toHaveBeenCalled();

    const formFieldProps = form.find("PackForm").prop("fields");

    expect(formFieldProps.name).toMatchObject({
      error: "Title field must be completed",
    });
  });

  it("calls the handleSubmit prop when a valid form is submitted", () => {
    const handleSubmitSpy = jest.fn();
    const form = mount(<PackForm handleSubmit={handleSubmitSpy} />).find(
      "form"
    );
    const nameField = form.find("InputField").find({ name: "name" });

    fillInFormInput(nameField, "Mac OS Attacks");

    form.simulate("submit");

    expect(handleSubmitSpy).toHaveBeenCalled();
  });
});
