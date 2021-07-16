import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import { scheduledQueryStub } from "test/stubs";
import { fillInFormInput } from "test/helpers";
import ScheduledQueriesListWrapper from "./index";

const scheduledQueries = [
  scheduledQueryStub,
  { ...scheduledQueryStub, id: 100, name: "mac hosts" },
];
const defaultProps = {
  onRemoveScheduledQueries: noop,
  onScheduledQueryFormSubmit: noop,
  onSelectScheduledQuery: noop,
  onDblClickScheduledQuery: noop,
  scheduledQueries,
};

describe("ScheduledQueriesListWrapper - component", () => {
  it('renders the "Remove Query" button when queries have been selected', () => {
    const component = mount(<ScheduledQueriesListWrapper {...defaultProps} />);

    component.find("Checkbox").last().find("input").simulate("change");

    const addQueryBtn = component
      .find("Button")
      .find({ children: "Add New Query" });
    const removeQueryBtn = component
      .find("Button")
      .find({ children: "Remove" });

    expect(addQueryBtn.length).toEqual(0);
    expect(removeQueryBtn.length).toBeGreaterThan(0);
  });

  it("calls the onRemoveScheduledQueries prop", () => {
    const spy = jest.fn();
    const props = { ...defaultProps, onRemoveScheduledQueries: spy };
    const component = mount(<ScheduledQueriesListWrapper {...props} />);

    component
      .find(`.row-${scheduledQueryStub.id}`)
      .find("Checkbox")
      .find("input")
      .simulate("change");

    const removeQueryBtn = component
      .find("Button")
      .find({ children: "Remove" });

    removeQueryBtn.hostNodes().simulate("click");

    expect(spy).toHaveBeenCalledWith([scheduledQueryStub.id]);
  });

  // TODO: Test search and select all at TableContainer component level
});
