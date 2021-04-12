import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import { DropdownButton } from "./DropdownButton";

describe("DropdownButton - component", () => {
  it("calls the clicked item's onClick attribute", () => {
    const optionSpy = jest.fn();
    const dropdownOptions = [
      { label: "btn1", onClick: noop },
      { label: "btn2", onClick: optionSpy },
    ];
    const component = mount(
      <DropdownButton options={dropdownOptions}>New Button</DropdownButton>
    );

    component.find("button.dropdown-button").simulate("click");
    expect(component.state().isOpen).toEqual(true);

    component
      .find("li.dropdown-button__option")
      .last()
      .find("Button")
      .simulate("click");
    expect(optionSpy).toHaveBeenCalled();
  });
});
