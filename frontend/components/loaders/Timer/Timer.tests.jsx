import React from "react";
import { mount } from "enzyme";

import Timer from "./Timer";

describe("Timer - component", () => {
  it("renders with proper time", () => {
    const timer1 = mount(<Timer totalMilliseconds={1000} />);
    const elem1 = timer1.find(".kolide-timer");

    expect(elem1.text()).toEqual("00:00:01");

    const timer2 = mount(<Timer totalMilliseconds={60000} />);
    const elem2 = timer2.find(".kolide-timer");

    expect(elem2.text()).toEqual("00:01:00");

    const timer3 = mount(<Timer totalMilliseconds={3600000} />);
    const elem3 = timer3.find(".kolide-timer");

    expect(elem3.text()).toEqual("01:00:00");
  });
});
