import React from "react";
import { mount } from "enzyme";

import PlatformIcon from "./PlatformIcon";

describe("PlatformIcon - component", () => {
  it("renders", () => {
    expect(mount(<PlatformIcon name="linux" />).length).toEqual(1);
  });

  it("renders text if no icon", () => {
    const component = mount(<PlatformIcon name="All" />);

    expect(component.find(".fleeticon-single-host").length).toEqual(1);
  });
});
