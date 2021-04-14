import React from "react";
import { mount } from "enzyme";

import Dropdown from "components/forms/fields/Dropdown";
import { fillInFormInput } from "test/helpers";

describe("Dropdown - component", () => {
  const options = [
    { text: "Users", value: "users" },
    { text: "Groups", value: "groups" },
  ];

  const props = {
    name: "my-select",
    options,
  };

  it("renders the dropdown", () => {
    const component = mount(<Dropdown {...props} />);
    const dropdownSelect = component.find("Select");

    expect(dropdownSelect).toBeTruthy();
  });

  it("selects a value from dropdown", () => {
    const onChangeSpy = jest.fn();
    const component = mount(<Dropdown {...props} onChange={onChangeSpy} />);
    const inputNode = component.find("input");

    fillInFormInput(inputNode, "users");
    component.find(".Select-option").first().simulate("mousedown");

    expect(onChangeSpy).toHaveBeenCalledWith("users");
  });
});
