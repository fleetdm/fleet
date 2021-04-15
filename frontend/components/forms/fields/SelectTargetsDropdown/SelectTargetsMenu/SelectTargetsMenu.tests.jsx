import React from "react";
import PropTypes from "prop-types";
import { mount } from "enzyme";
import { noop } from "lodash";

import SelectTargetsMenuWrapper from "components/forms/fields/SelectTargetsDropdown/SelectTargetsMenu";
import Test from "test";

const DummyOption = (props) => {
  return <div>{props.children}</div>;
};

DummyOption.propTypes = { children: PropTypes.node };

describe("SelectTargetsMenu - component", () => {
  let onMoreInfoClick = noop;
  const moreInfoTarget = undefined;
  const handleBackToResults = noop;
  const defaultProps = {
    focusedOption: undefined,
    instancePrefix: "",
    onFocus: noop,
    onOptionRef: noop,
    onSelect: noop,
    optionComponent: DummyOption,
  };

  describe("rendering", () => {
    it("renders", () => {
      const SelectTargetsMenu = SelectTargetsMenuWrapper(
        onMoreInfoClick,
        moreInfoTarget,
        handleBackToResults
      );
      const Component = mount(<SelectTargetsMenu {...defaultProps} />);

      expect(Component.length).toEqual(1);
    });

    it("renders no target text", () => {
      const SelectTargetsMenu = SelectTargetsMenuWrapper(
        onMoreInfoClick,
        moreInfoTarget,
        handleBackToResults
      );
      const Component = mount(<SelectTargetsMenu {...defaultProps} />);
      const componentText = Component.text();

      expect(componentText).toContain("Unable to find any matching labels.");
      expect(componentText).toContain("Unable to find any matching hosts.");
    });

    it("renders a target option component for each target", () => {
      const SelectTargetsMenu = SelectTargetsMenuWrapper(
        onMoreInfoClick,
        moreInfoTarget,
        handleBackToResults
      );
      const options = [Test.Stubs.labelStub, Test.Stubs.hostStub];
      const props = { ...defaultProps, options };
      const Component = mount(<SelectTargetsMenu {...props} />);
      const TargetOption = Component.find("TargetOption");

      expect(TargetOption.length).toEqual(options.length);

      expect(options).toContainEqual(TargetOption.first().prop("target"));
      expect(options).toContainEqual(TargetOption.last().prop("target"));
    });
  });

  describe("clicking a target", () => {
    it("calls the onMoreInfoClick function", () => {
      const spy = jest.fn();
      onMoreInfoClick = (t) => {
        return () => spy(t);
      };
      const SelectTargetsMenu = SelectTargetsMenuWrapper(
        onMoreInfoClick,
        moreInfoTarget,
        handleBackToResults
      );
      const options = [Test.Stubs.labelStub];
      const props = { ...defaultProps, options };
      const Component = mount(<SelectTargetsMenu {...props} />);
      const TargetOption = Component.find("TargetOption");

      TargetOption.find(".target-option__target-content").simulate("click");

      expect(spy).toHaveBeenCalledWith(Test.Stubs.labelStub);
    });

    it("calls the onSelect prop when the add button is clicked", () => {
      const spy = jest.fn();
      const SelectTargetsMenu = SelectTargetsMenuWrapper(
        onMoreInfoClick,
        moreInfoTarget,
        handleBackToResults
      );
      const options = [Test.Stubs.labelStub];
      const props = { ...defaultProps, onSelect: spy, options };
      const Component = mount(<SelectTargetsMenu {...props} />);
      const TargetOption = Component.find("TargetOption");

      TargetOption.find(".target-option__add-btn").simulate("click");

      expect(spy).toHaveBeenCalled();
    });
  });
});
