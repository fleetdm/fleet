import React from "react";
import { render, screen } from "@testing-library/react";

import ScheduledQueriesSection from "components/side_panels/PackDetailsSidePanel/ScheduledQueriesSection";
import { scheduledQueryStub } from "test/stubs";

describe("ScheduledQueriesSection - component", () => {
  it("links the query name to the show query route", () => {
    const scheduledQuery = { ...scheduledQueryStub, query_id: 55 };
    render(<ScheduledQueriesSection scheduledQueries={[scheduledQuery]} />);

    expect(screen.getByText("Get all users")).toBeInTheDocument();
    expect(screen.getByText("Get all users").closest("a")).toBeInTheDocument();
  });
});
