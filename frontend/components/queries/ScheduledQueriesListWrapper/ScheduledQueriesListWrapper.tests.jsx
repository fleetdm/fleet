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
      .find({ children: ["Remove ", "query"] });

    expect(addQueryBtn.length).toEqual(0);
    expect(removeQueryBtn.length).toBeGreaterThan(0);
  });

  it("calls the onRemoveScheduledQueries prop", () => {
    const spy = jest.fn();
    const props = { ...defaultProps, onRemoveScheduledQueries: spy };
    const component = mount(<ScheduledQueriesListWrapper {...props} />);

    component
      .find("Checkbox")
      .find({ name: `scheduled-query-checkbox-${scheduledQueryStub.id}` })
      .find("input")
      .simulate("change");

    const removeQueryBtn = component
      .find("Button")
      .find({ children: ["Remove ", "query"] });

    removeQueryBtn.hostNodes().simulate("click");

    expect(spy).toHaveBeenCalledWith([scheduledQueryStub.id]);
  });

  it("filters queries", () => {
    const component = mount(<ScheduledQueriesListWrapper {...defaultProps} />);

    const searchQueriesInput = component.find({ name: "search-queries" });
    let QueriesList = component.find("ScheduledQueriesList");

    expect(QueriesList.prop("scheduledQueries")).toEqual(scheduledQueries);

    fillInFormInput(searchQueriesInput, "something that does not match");

    QueriesList = component.find("ScheduledQueriesList");
    expect(QueriesList.prop("scheduledQueries")).toEqual([]);
  });

  it("allows selecting all scheduled queries at once", () => {
    const allScheduledQueryIDs = scheduledQueries.map((sq) => sq.id);
    const component = mount(<ScheduledQueriesListWrapper {...defaultProps} />);
    const selectAllCheckbox = component.find({
      name: "select-all-scheduled-queries",
    });

    selectAllCheckbox.hostNodes().simulate("change");

    expect(component.state("checkedScheduledQueryIDs")).toEqual(
      allScheduledQueryIDs
    );

    selectAllCheckbox.hostNodes().simulate("change");

    expect(component.state("checkedScheduledQueryIDs")).toEqual([]);
  });
});
