import React from "react";
import { noop } from "lodash";
import { render, screen, within } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import FilterPill from "./FilterPill";

describe("Filter Pill Component", () => {
  it("renders the pill text", () => {
    render(<FilterPill label="Test Pill" onClear={noop} />);

    expect(screen.getByText("Test Pill")).toBeInTheDocument();
  });

  it("renders icon properly", () => {
    render(<FilterPill label="Test Pill" icon="policy" onClear={noop} />);

    expect(
      within(screen.getByRole("status")).getByTestId("policy-icon")
    ).toBeInTheDocument();
  });

  it("renders a passed in string tooltip", () => {
    render(
      <FilterPill
        label="Test Pill"
        tooltipDescription="Test Tooltip"
        onClear={noop}
      />
    );

    expect(screen.getByText("Test Tooltip")).toBeInTheDocument();
  });

  it("renders a passed in ReactNode tooltip", () => {
    render(
      <FilterPill
        label="Test Pill"
        tooltipDescription={<p>This is a ReactNode</p>}
        onClear={noop}
      />
    );

    expect(screen.getByText("This is a ReactNode")).toBeInTheDocument();
  });

  it("calls the onCancel callback when a user clicks on the remove button", async () => {
    const spy = jest.fn();

    const { user } = renderWithSetup(
      <FilterPill label="Test Pill" onClear={spy} />
    );

    await user.click(
      within(screen.getByRole("button")).getByTestId("close-icon")
    );

    expect(spy).toHaveBeenCalled();
  });
});
