import React from "react";
import { mount } from "enzyme";

import Breadcrumbs from "pages/RegistrationPage/Breadcrumbs";

describe("Breadcrumbs - component", () => {
  it("renders 3 Button components", () => {
    const component = mount(<Breadcrumbs page={1} />);
    expect(component.find("button").length).toEqual(3);
  });

  it("renders page 1 Button as active when the page prop is 1", () => {
    const component = mount(<Breadcrumbs page={1} />);
    const page1Btn = component.find("button.registration-breadcrumbs__page--1");
    const page2Btn = component.find("button.registration-breadcrumbs__page--2");
    const page3Btn = component.find("button.registration-breadcrumbs__page--3");

    expect(page1Btn.prop("className")).toContain(
      "registration-breadcrumbs__page--active"
    );
    expect(page2Btn.prop("className")).not.toContain(
      "registration-breadcrumbs__page--active"
    );
    expect(page3Btn.prop("className")).not.toContain(
      "registration-breadcrumbs__page--active"
    );
  });

  it("calls the onClick prop with the page number when clicked", () => {
    const onClickSpy = jest.fn();
    const component = mount(<Breadcrumbs page={1} onClick={onClickSpy} />);
    const page1Btn = component.find("button.registration-breadcrumbs__page--1");
    const page2Btn = component.find("button.registration-breadcrumbs__page--2");
    const page3Btn = component.find("button.registration-breadcrumbs__page--3");

    page1Btn.simulate("click");

    expect(onClickSpy).toHaveBeenCalledWith(1);

    page2Btn.simulate("click");

    expect(onClickSpy).toHaveBeenCalledWith(2);

    page3Btn.simulate("click");

    expect(onClickSpy).toHaveBeenCalledWith(3);
  });
});
