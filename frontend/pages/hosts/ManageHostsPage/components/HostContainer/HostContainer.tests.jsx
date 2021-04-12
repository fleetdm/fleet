import React from "react";
import { shallow } from "enzyme";

import HostContainer from "./HostContainer";

const allHostsLabel = {
  id: 1,
  display_text: "All Hosts",
  slug: "all-hosts",
  type: "all",
  count: 0,
};

describe("HostsContainer - component", () => {
  const props = {
    selectedLabel: allHostsLabel,
  };

  it("displays getting started text if no hosts available", () => {
    const page = shallow(
      <HostContainer
        {...props}
        selectedFilter={"all-hosts"}
        selectedLabel={allHostsLabel}
      />
    );

    expect(page.find("h2").text()).toEqual(
      "Get started adding hosts to Fleet."
    );
  });

  it("renders the DataTable if there are hosts", () => {
    const page = shallow(<HostContainer {...props} />);

    expect(page.find("DataTable").length).toEqual(1);
  });
});
