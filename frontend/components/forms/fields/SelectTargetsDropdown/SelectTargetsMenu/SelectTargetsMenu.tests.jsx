import React from "react";
import PropTypes from "prop-types";
import { fireEvent, render, screen } from "@testing-library/react";

import SelectTargetsMenuWrapper from "components/forms/fields/SelectTargetsDropdown/SelectTargetsMenu";
import Test from "test";

const DummyOption = (props) => {
  return <div>{props.children}</div>;
};

DummyOption.propTypes = { children: PropTypes.node };

describe("SelectTargetsMenu - component", () => {
  let onMoreInfoClick = jest.fn();
  const moreInfoTarget = undefined;
  const handleBackToResults = jest.fn();
  const defaultProps = {
    focusedOption: undefined,
    instancePrefix: "",
    onFocus: jest.fn(),
    onOptionRef: jest.fn(),
    onSelect: jest.fn(),
    optionComponent: DummyOption,
  };

  it("renders", () => {
    const SelectTargetsMenu = SelectTargetsMenuWrapper(
      onMoreInfoClick,
      moreInfoTarget,
      handleBackToResults
    );
    const { container } = render(<SelectTargetsMenu {...defaultProps} />);

    expect(container).not.toBeNull();
  });

  it("renders no target text", () => {
    const SelectTargetsMenu = SelectTargetsMenuWrapper(
      onMoreInfoClick,
      moreInfoTarget,
      handleBackToResults
    );

    render(<SelectTargetsMenu {...defaultProps} />);

    expect(
      screen.getByText("Unable to find any matching labels.")
    ).toBeInTheDocument();
    expect(
      screen.getByText("Unable to find any matching hosts.")
    ).toBeInTheDocument();
  });

  it("renders a target option component for each target", () => {
    const SelectTargetsMenu = SelectTargetsMenuWrapper(
      onMoreInfoClick,
      moreInfoTarget,
      handleBackToResults
    );
    const options = [Test.Stubs.labelStub, Test.Stubs.hostStub];
    const props = { ...defaultProps, options };
    const { container } = render(<SelectTargetsMenu {...props} />);

    const TargetOption = container.querySelectorAll(".target-option__wrapper");

    expect(TargetOption.length).toEqual(options.length);

    expect(container.querySelectorAll(".is-label").length).toEqual(1);
    expect(container.querySelectorAll(".is-host").length).toEqual(1);
  });

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

    render(<SelectTargetsMenu {...props} />);

    fireEvent.click(screen.getByText("All hosts"));

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

    render(<SelectTargetsMenu {...props} />);

    fireEvent.click(screen.getAllByRole("button")[1]);

    expect(spy).toHaveBeenCalled();
  });
});
