import React from "react";
import { noop } from "lodash";
import { render, screen, within, waitFor } from "@testing-library/react";
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

  it("renders a passed in string tooltip", async () => {
    const { user } = renderWithSetup(
      <FilterPill
        label="Test Pill"
        tooltipDescription="Test Tooltip"
        onClear={noop}
      />
    );

    await user.hover(screen.getByText("Test Pill"));
    await waitFor(() => {
      const tooltip = screen.getByText("Test Tooltip");
      expect(tooltip).toBeInTheDocument();
    });
  });

  it("renders a passed in ReactNode tooltip", async () => {
    const { user } = renderWithSetup(
      <FilterPill
        label="Test Pill"
        tooltipDescription={<p>This is a ReactNode</p>}
        onClear={noop}
      />
    );

    await user.hover(screen.getByText("Test Pill"));
    await waitFor(() => {
      const tooltip = screen.getByText("This is a ReactNode");
      expect(tooltip).toBeInTheDocument();
    });
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
