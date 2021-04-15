import React from "react";
import { mount } from "enzyme";

import ScheduledQueriesSection from "components/side_panels/PackDetailsSidePanel/ScheduledQueriesSection";
import { scheduledQueryStub } from "test/stubs";

describe("ScheduledQueriesSection - component", () => {
  it("links the query name to the show query route", () => {
    const scheduledQuery = { ...scheduledQueryStub, query_id: 55 };
    const Component = mount(
      <ScheduledQueriesSection scheduledQueries={[scheduledQuery]} />
    );
    const Link = Component.find("Link");

    expect(Link.prop("to")).toEqual(`/queries/${scheduledQuery.query_id}`);
  });
});
