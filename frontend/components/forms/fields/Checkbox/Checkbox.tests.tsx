import React from "react";
import { mount } from "enzyme";

import Checkbox from "./Checkbox";

describe("Checkbox - component", () => {
  it("renders", () => {
    expect(mount(<Checkbox />)).toBeTruthy();
  });

  it('calls the "onChange" handler when changed', () => {
    const onCheckedComponentChangeSpy = jest.fn();
    const onUncheckedComponentChangeSpy = jest.fn();

    const checkedComponent = mount(
      <Checkbox name="checkbox" onChange={onCheckedComponentChangeSpy} value />
    ).find("input");

    const uncheckedComponent = mount(
      <Checkbox
        name="checkbox"
        onChange={onUncheckedComponentChangeSpy}
        value={false}
      />
    ).find("input");

    checkedComponent.simulate("change");
    uncheckedComponent.simulate("change");

    expect(onCheckedComponentChangeSpy).toHaveBeenCalledWith(false);
    expect(onUncheckedComponentChangeSpy).toHaveBeenCalledWith(true);
  });
});
