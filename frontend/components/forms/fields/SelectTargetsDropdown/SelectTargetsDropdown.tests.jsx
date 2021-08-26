import React from "react";
import { mount, shallow } from "enzyme";
import nock from "nock";
import { noop } from "lodash";

import SelectTargetsDropdown from "components/forms/fields/SelectTargetsDropdown";
import Test from "test";

describe("SelectTargetsDropdown - component", () => {
  beforeEach(() => Test.Mocks.targetMock().persist());

  const defaultProps = {
    disabled: false,
    label: "Select Targets",
    onFetchTargets: noop,
    onSelect: noop,
    selectedTargets: [],
    targetsCount: 0,
    queryId: 1,
  };
  afterEach(() => nock.cleanAll());

  it("sets default state", () => {
    const DefaultComponent = mount(<SelectTargetsDropdown {...defaultProps} />);
    expect(DefaultComponent.state()).toEqual({
      isEmpty: false,
      isLoadingTargets: true,
      moreInfoTarget: null,
      query: "",
      targets: [],
    });
  });

  describe("rendering", () => {
    it("renders", () => {
      const DefaultComponent = mount(
        <SelectTargetsDropdown {...defaultProps} />
      );
      expect(DefaultComponent.length).toEqual(
        1,
        "Expected component to render"
      );
    });

    it("renders the SelectTargetsInput", () => {
      const DefaultComponent = mount(
        <SelectTargetsDropdown {...defaultProps} />
      );
      const SelectTargetsInput = DefaultComponent.find("SelectTargetsInput");

      expect(SelectTargetsInput.length).toEqual(
        1,
        "Expected SelectTargetsInput to render"
      );
    });

    it("renders a label when passed as a prop", () => {
      const DefaultComponent = mount(
        <SelectTargetsDropdown {...defaultProps} />
      );
      const noLabelProps = { ...defaultProps, label: undefined };
      const ComponentWithoutLabel = mount(
        <SelectTargetsDropdown {...noLabelProps} />
      );
      const Label = DefaultComponent.find(".target-select__label");
      const NoLabel = ComponentWithoutLabel.find(".target-select__label");

      expect(Label.length).toEqual(1, "Expected label to render");
      expect(NoLabel.length).toEqual(0, "Expected label to not render");
    });

    it("renders the error when passed as a prop", () => {
      const DefaultComponent = mount(
        <SelectTargetsDropdown {...defaultProps} />
      );
      const errorProps = { ...defaultProps, error: "You can't do this!" };
      const ErrorComponent = mount(<SelectTargetsDropdown {...errorProps} />);
      const Error = ErrorComponent.find(".target-select__label--error");
      const NoError = DefaultComponent.find(".target-select__label--error");

      expect(Error.length).toEqual(1, "Expected error to render");
      expect(NoError.length).toEqual(0, "Expected error to not render");
    });

    it("renders the target count", () => {
      const DefaultComponent = mount(
        <SelectTargetsDropdown {...defaultProps} />
      );
      const targetCountProps = { ...defaultProps, targetsCount: 10 };
      const TargetCountComponent = mount(
        <SelectTargetsDropdown {...targetCountProps} />
      );

      expect(DefaultComponent.text()).toContain("0 unique hosts");
      expect(TargetCountComponent.text()).toContain("10 unique hosts");
    });
  });

  describe("#fetchTargets", () => {
    const apiResponseWithTargets = {
      targets: {
        hosts: [],
        labels: [Test.Stubs.labelStub],
        teams: [],
      },
    };
    const apiResponseWithoutTargets = {
      targets: {
        hosts: [],
        labels: [],
        teams: [],
      },
    };
    const defaultSelectedTargets = { hosts: [], labels: [], teams: [] };
    const defaultParams = {
      query: "",
      query_id: 1,
      selected: defaultSelectedTargets,
    };
    const expectedApiClientResponseWithTargets = {
      targets: [{ ...Test.Stubs.labelStub, target_type: "labels" }],
    };

    it("calls the onFetchTargets prop", () => {
      const onFetchTargets = jest.fn();
      const props = { ...defaultProps, onFetchTargets };

      nock.cleanAll();
      Test.Mocks.targetMock(defaultParams, apiResponseWithTargets).persist();

      const Component = shallow(<SelectTargetsDropdown {...props} />);
      const node = Component.instance();

      expect.assertions(1);
      return node.fetchTargets().then(() => {
        expect(onFetchTargets).toHaveBeenCalledWith(
          "",
          expectedApiClientResponseWithTargets
        );
      });
    });

    it("does not call the onFetchTargets prop when the component is not mounted", () => {
      const onFetchTargets = jest.fn();
      const props = { ...defaultProps, onFetchTargets };
      const Component = shallow(<SelectTargetsDropdown {...props} />);
      const node = Component.instance();

      node.mounted = false;

      expect(node.fetchTargets()).toEqual(false);
      expect(onFetchTargets).not.toHaveBeenCalled();
    });

    it("sets state correctly when no targets are returned", () => {
      const Component = mount(<SelectTargetsDropdown {...defaultProps} />);
      const node = Component.instance();

      Test.Mocks.targetMock(defaultParams, apiResponseWithoutTargets);
      expect.assertions(3);
      return node.fetchTargets().then(() => {
        expect(Component.state("isEmpty")).toEqual(true);
        expect(Component.state("targets")).toEqual([{}]);
        expect(Component.state("isLoadingTargets")).toEqual(false);
      });
    });

    it("returns the query", () => {
      const query = "select * from users";
      const Component = mount(<SelectTargetsDropdown {...defaultProps} />);
      const node = Component.instance();

      Test.Mocks.targetMock({ ...defaultParams, query });

      expect.assertions(1);
      return node.fetchTargets(query).then((q) => {
        expect(q).toEqual(query);
      });
    });
  });
});
