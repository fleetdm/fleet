import React from "react";
import { noop } from "lodash";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import Radio from "./Radio";

describe("Radio - component", () => {
  it("renders the radio label text from the label prop", () => {
    render(
      <Radio
        checked
        label="Radio Label"
        value="radioValue"
        id="test-radio"
        onChange={noop}
      />
    );

    const labelText = screen.getByText("Radio Label");
    expect(labelText).toBeInTheDocument();
  });

  it("passes the radio input value when checked", async () => {
    const user = userEvent.setup();
    const changeHandlerSpy = jest.fn();

    render(
      <Radio
        label="Radio Label"
        value="radioValue"
        id="test-radio"
        onChange={changeHandlerSpy}
      />
    );

    const radio = screen.getByRole("radio", { name: "Radio Label" });
    await user.click(radio);

    expect(changeHandlerSpy).toHaveBeenCalled();
    expect(changeHandlerSpy).toHaveBeenCalledWith("radioValue");
  });

  it("renders the correct selected state from checked prop", () => {
    render(
      <Radio
        checked
        label="Radio Label"
        value="radioValue"
        id="test-radio"
        onChange={noop}
      />
    );

    const radio = screen.getByRole("radio", { name: "Radio Label" });
    expect(radio).toBeChecked();
  });

  it("renders the correct disabled state from disabled prop", () => {
    render(
      <Radio
        disabled
        label="Radio Label"
        value="radioValue"
        id="test-radio"
        onChange={noop}
        testId="radio-input"
      />
    );

    const radio = screen.getByRole("radio", { name: "Radio Label" });
    expect(radio).toBeDisabled();

    // Also adds a disabled class to the componet
    const radioComponent = screen.getByTestId("radio-input");
    expect(radioComponent).toHaveClass("disabled");
  });

  it("render a tooltip from the tooltip prop", async () => {
    render(
      <Radio
        disabled
        label="Radio Label"
        value="radioValue"
        id="test-radio"
        onChange={noop}
        tooltip="A Test Radio Tooltip"
      />
    );

    await fireEvent.mouseEnter(screen.getByText("Radio Label"));
    const tooltip = screen.getByText("A Test Radio Tooltip");
    expect(tooltip).toBeInTheDocument();
  });

  it("adds the custom class name from the className prop", () => {
    render(
      <Radio
        disabled
        label="Radio Label"
        value="radioValue"
        id="test-radio"
        onChange={noop}
        className="radio-button"
        testId="radio-input"
      />
    );

    const radioComponent = screen.getByTestId("radio-input");
    expect(radioComponent).toHaveClass("radio-button");
  });
});
