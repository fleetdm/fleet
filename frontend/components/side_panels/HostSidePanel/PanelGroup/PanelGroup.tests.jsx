import React from "react";
import { mount } from "enzyme";

import PanelGroup from "./PanelGroup";

describe("PanelGroup - component", () => {
  const validPanelGroupItems = [
    { type: "all", display_text: "All Hosts", hosts_count: 20 },
    { type: "platform", display_text: "MAC OS", hosts_count: 10 },
  ];

  const component = mount(<PanelGroup groupItems={validPanelGroupItems} />);

  it("renders a PanelGroupItem for each group item", () => {
    const panelGroupItems = component.find("PanelGroupItem");

    expect(panelGroupItems.length).toEqual(2);
  });
});
