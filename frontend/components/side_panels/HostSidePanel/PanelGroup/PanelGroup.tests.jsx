import React from "react";
import { render, screen } from "@testing-library/react";

import PanelGroup from "./PanelGroup";

describe("PanelGroup - component", () => {
  const validPanelGroupItems = [
    { type: "all", display_text: "All Hosts", hosts_count: 20 },
    { type: "platform", display_text: "MAC OS", hosts_count: 10 },
  ];

  render(<PanelGroup groupItems={validPanelGroupItems} />);

  it("renders a PanelGroupItem for each group item", () => {
    const panelGroupItems = screen.queryAllByRole("button");

    expect(panelGroupItems.length).toEqual(2);
  });
});
