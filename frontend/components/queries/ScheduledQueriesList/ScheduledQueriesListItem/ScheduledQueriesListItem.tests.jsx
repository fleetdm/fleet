import React from "react";
import { mount, shallow } from "enzyme";
import { noop } from "lodash";

import { scheduledQueryStub } from "test/stubs";
import ScheduledQueriesListItem from "./index";

const defaultProps = {
  checked: false,
  onCheck: noop,
  onSelect: noop,
  onDblClick: noop,
  scheduledQuery: scheduledQueryStub,
};

describe("ScheduledQueriesListItem - component", () => {
  it("renders the scheduled query data", () => {
    const component = mount(<ScheduledQueriesListItem {...defaultProps} />);
    expect(component.text()).toContain(scheduledQueryStub.query_name);
    expect(component.text()).toContain(scheduledQueryStub.interval);
    expect(component.text()).toContain(scheduledQueryStub.shard);
  });

  it("renders when the platform attribute is null", () => {
    const scheduledQuery = { ...scheduledQueryStub, platform: null };
    const component = mount(
      <ScheduledQueriesListItem
        checked={false}
        scheduledQuery={scheduledQuery}
        {...defaultProps}
      />
    );
    expect(component.text()).toContain(scheduledQueryStub.query_name);
    expect(component.text()).toContain(scheduledQueryStub.interval);
    expect(component.text()).toContain(scheduledQueryStub.shard);
  });

  it("renders the converted platform attributes", () => {
    const scheduledQuery = {
      ...scheduledQueryStub,
      platform: "darwin,linux,all,windows",
    };
    const component = mount(
      <ScheduledQueriesListItem
        checked={false}
        {...defaultProps}
        scheduledQuery={scheduledQuery}
      />
    );
    expect(component.text()).toContain("macOS");
    expect(component.text()).toContain("Linux");
    expect(component.text()).toContain("All");
    expect(component.text()).toContain("Windows");
  });

  it("renders the platform attributes when there are no conversions", () => {
    const scheduledQuery = {
      ...scheduledQueryStub,
      platform: "darwin,freebsd,  bar, foo",
    };
    const component = mount(
      <ScheduledQueriesListItem
        checked={false}
        {...defaultProps}
        scheduledQuery={scheduledQuery}
      />
    );
    expect(component.text()).toContain("macOS");
    expect(component.text()).toContain("freebsd");
    expect(component.text()).toContain("bar");
    expect(component.text()).toContain("foo");
  });

  it("renders a Checkbox component", () => {
    const component = shallow(<ScheduledQueriesListItem {...defaultProps} />);
    expect(component.find("Checkbox").length).toEqual(1);
  });

  it("calls the onCheck prop when a checkbox is changed", () => {
    const onCheckSpy = jest.fn();
    const props = { ...defaultProps, onCheck: onCheckSpy };
    const component = mount(<ScheduledQueriesListItem {...props} />);
    const checkbox = component.find("Checkbox").first();

    checkbox.find("input").simulate("change");

    expect(onCheckSpy).toHaveBeenCalledWith(true, scheduledQueryStub.id);
  });

  it("calls the onSelect prop when a list item is selected", () => {
    const spy = jest.fn();
    const props = { ...defaultProps, onSelect: spy };
    const component = shallow(<ScheduledQueriesListItem {...props} />);
    const tableRow = component.find("ClickableTableRow");

    tableRow.simulate("click");

    expect(spy).toHaveBeenCalledWith(scheduledQueryStub);
  });

  it("calls the onDblClick prop when a list item is double clicked", () => {
    const spy = jest.fn();
    const props = { ...defaultProps, onDblClick: spy };
    const component = shallow(<ScheduledQueriesListItem {...props} />);
    const tableRow = component.find("ClickableTableRow");

    tableRow.simulate("doubleclick");

    expect(spy).toHaveBeenCalledWith(scheduledQueryStub.query_id);
  });

  describe("renders the appropriate query type icon", () => {
    const query = { ...scheduledQueryStub, removed: null };
    const props = { ...defaultProps, scheduledQuery: query };
    let component = shallow(<ScheduledQueriesListItem {...props} />);

    expect(component.find("FleetIcon").last().props().name).toEqual("camera");

    query.snapshot = false;
    query.removed = false;
    component = shallow(<ScheduledQueriesListItem {...props} />);
    expect(component.find("FleetIcon").last().props().name).toEqual(
      "bold-plus"
    );

    query.snapshot = false;
    query.removed = null;
    component = shallow(<ScheduledQueriesListItem {...props} />);
    expect(component.find("FleetIcon").last().props().name).toEqual(
      "plus-minus"
    );

    query.snapshot = false;
    query.removed = true;
    component = shallow(<ScheduledQueriesListItem {...props} />);
    expect(component.find("FleetIcon").last().props().name).toEqual(
      "plus-minus"
    );

    query.snapshot = null;
    query.removed = true;
    component = shallow(<ScheduledQueriesListItem {...props} />);
    expect(component.find("FleetIcon").last().props().name).toEqual(
      "plus-minus"
    );
  });
});
