import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import { hostStub, labelStub } from "test/stubs";
import TargetOption from "./TargetOption";

describe("TargetOption - component", () => {
  it("renders a label option for label targets", () => {
    const count = 5;
    const component = mount(
      <TargetOption onMoreInfoClick={noop} target={{ ...labelStub, count }} />
    );
    expect(component.find(".is-label").length).toEqual(1);
    expect(component.text()).toContain(`${count} hosts`);
  });

  it("renders a host option for host targets", () => {
    const component = mount(
      <TargetOption
        onMoreInfoClick={noop}
        target={{ ...hostStub, platform: "windows" }}
      />
    );
    expect(component.find(".is-host").length).toEqual(1);
    expect(component.find("i.fleeticon-windows").length).toEqual(1);
    expect(component.text()).toContain(hostStub.primary_ip);
  });

  it("calls the onSelect prop when + icon button is clicked", () => {
    const onSelectSpy = jest.fn();
    const component = mount(
      <TargetOption
        onMoreInfoClick={noop}
        onSelect={onSelectSpy}
        target={hostStub}
      />
    );
    component.find(".target-option__add-btn").simulate("click");
    expect(onSelectSpy).toHaveBeenCalled();
  });

  it("calls the onMoreInfoClick prop when the item content is clicked", () => {
    const onMoreInfoClickSpy = jest.fn();
    const onMoreInfoClick = () => {
      return onMoreInfoClickSpy;
    };
    const component = mount(
      <TargetOption onMoreInfoClick={onMoreInfoClick} target={hostStub} />
    );
    component.find(".target-option__target-content").simulate("click");
    expect(onMoreInfoClickSpy).toHaveBeenCalled();
  });
});
