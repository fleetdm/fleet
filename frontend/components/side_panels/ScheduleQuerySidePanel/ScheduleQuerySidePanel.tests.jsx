import React from "react";
import { render } from "@testing-library/react";

import ScheduleQuerySidePanel from "./ScheduleQuerySidePanel";

describe("ScheduleQuerySidePanel - component", () => {
  it("renders SearchPackQuery", () => {
    const { container } = render(<ScheduleQuerySidePanel />);

    expect(container.querySelectorAll(".search-pack-query").length).toEqual(1);
  });
});
