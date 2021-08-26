import React from "react";
import { mount } from "enzyme";

import FleetIcon from "./FleetIcon";

describe("FleetIcon - component", () => {
  it("renders", () => {
    expect(mount(<FleetIcon name="success-check" />)).toBeTruthy();
  });
});
