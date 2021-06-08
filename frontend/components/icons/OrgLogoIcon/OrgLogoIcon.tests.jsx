import React from "react";
import { mount } from "enzyme";

import fleetAvatar from "../../../../assets/images/fleet-avatar-24x24@2x.png";
import OrgLogoIcon from "./OrgLogoIcon";

describe("OrgLogoIcon - component", () => {
  it("renders the Kolide Logo by default", () => {
    const component = mount(<OrgLogoIcon />);

    expect(component.state("imageSrc")).toEqual(fleetAvatar);
  });

  it("renders the image source when it is valid", () => {
    const component = mount(<OrgLogoIcon src="/assets/images/avatar.svg" />);

    expect(component.state("imageSrc")).toEqual("/assets/images/avatar.svg");
  });
});
