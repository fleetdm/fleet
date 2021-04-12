import React from "react";
import { mount } from "enzyme";

import PanelGroupItem from "./PanelGroupItem";

describe("PanelGroupItem - component", () => {
  const id = 0;
  const validPanelGroupItem = {
    count: 20,
    display_text: "All Hosts",
    type: "all",
    id,
  };
  const validStatusGroupItem = {
    count: 111,
    display_text: "Online Hosts",
    id: "online",
    type: "status",
  };
  const statusLabels = {
    online_count: 20,
    loading_counts: false,
  };
  const loadingStatusLabels = {
    online_count: 20,
    loading_counts: true,
  };

  const labelComponent = mount(
    <PanelGroupItem item={validPanelGroupItem} statusLabels={statusLabels} />
  );

  const statusLabelComponent = mount(
    <PanelGroupItem
      item={validStatusGroupItem}
      statusLabels={statusLabels}
      type="status"
    />
  );

  const loadingStatusLabelComponent = mount(
    <PanelGroupItem
      item={validStatusGroupItem}
      statusLabels={loadingStatusLabels}
      type="status"
    />
  );

  it("renders the item text", () => {
    expect(labelComponent.text()).toContain(validPanelGroupItem.display_text);
  });

  it("renders the item count", () => {
    expect(labelComponent.text()).toContain(validPanelGroupItem.count);
    expect(statusLabelComponent.text()).not.toContain(
      validStatusGroupItem.count
    );
    expect(statusLabelComponent.text()).toContain(statusLabels.online_count);
    expect(loadingStatusLabelComponent.text()).not.toContain(
      statusLabels.online_count
    );
    expect(loadingStatusLabelComponent.text()).not.toContain(
      validPanelGroupItem.count
    );
  });
});
