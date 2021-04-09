import React from "react";
import { shallow } from "enzyme";

import WarningBanner from "components/WarningBanner/WarningBanner";

describe("WarningBanner - component", () => {
  it("renders empty when disabled", () => {
    const props = { shouldShowWarning: false, message: "message" };
    const component = shallow(<WarningBanner {...props} />);
    expect(component.html()).toBe(null);
  });
});
