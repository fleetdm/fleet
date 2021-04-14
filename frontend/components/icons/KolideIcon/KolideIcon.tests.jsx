import React from "react";
import { mount } from "enzyme";

import KolideIcon from "./KolideIcon";

describe("KolideIcon - component", () => {
  it("renders", () => {
    expect(mount(<KolideIcon name="success-check" />)).toBeTruthy();
  });
});
