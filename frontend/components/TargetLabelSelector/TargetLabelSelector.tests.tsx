import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import { noop } from "lodash";

import { ILabelSummary } from "interfaces/label";

import TargetLabelSelector, { ILabelConfig } from "./TargetLabelSelector";

const LABELS: ILabelSummary[] = [
  { id: 1, name: "label 1", label_type: "regular" },
  { id: 2, name: "label 2", label_type: "regular" },
];

const makeTab = (overrides: Partial<ILabelConfig> = {}): ILabelConfig => ({
  selectedLabels: {},
  onSelectLabel: noop,
  ...overrides,
});

const renderSelector = (
  props: Partial<React.ComponentProps<typeof TargetLabelSelector>> = {}
) =>
  render(
    <TargetLabelSelector
      selectedTargetType="Custom"
      onSelectTargetType={noop}
      labels={LABELS}
      includeConfig={makeTab({ showModeToggle: true, mode: "any" })}
      excludeConfig={makeTab()}
      emptyStateDescription="Add a label to target a group of hosts."
      onAddLabel={noop}
      {...props}
    />
  );

describe("TargetLabelSelector (tabbed) component", () => {
  it("renders Include and Exclude tabs with labels in Custom mode", () => {
    renderSelector();

    expect(screen.getByText("Include")).toBeVisible();
    expect(screen.getByText("Exclude")).toBeVisible();
    expect(screen.getByRole("checkbox", { name: "label 1" })).toBeVisible();
    expect(screen.getByRole("checkbox", { name: "label 2" })).toBeVisible();
  });

  it("does not render the custom tabs when the target type is 'All hosts'", () => {
    renderSelector({ selectedTargetType: "All hosts" });

    expect(screen.queryByText("Include")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("checkbox", { name: "label 1" })
    ).not.toBeInTheDocument();
  });

  it("renders the Any/All mode toggle on the include tab", () => {
    renderSelector();

    expect(screen.getByRole("radio", { name: "Any" })).toBeVisible();
    expect(screen.getByRole("radio", { name: "All" })).toBeVisible();
  });

  it("does not render the mode toggle on the exclude tab when showModeToggle is false", () => {
    renderSelector({ excludeConfig: makeTab({ showModeToggle: false }) });

    fireEvent.click(screen.getByText("Exclude"));

    expect(
      screen.queryByRole("radio", { name: "Any" })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("radio", { name: "All" })
    ).not.toBeInTheDocument();
  });

  it("renders selected labels as checked", () => {
    renderSelector({
      includeConfig: makeTab({
        showModeToggle: true,
        mode: "any",
        selectedLabels: { "label 1": true, "label 2": false },
      }),
    });

    expect(screen.getByRole("checkbox", { name: "label 1" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "label 2" })).not.toBeChecked();
  });

  it("disables a label in the include tab when it is selected in the exclude tab", () => {
    renderSelector({
      excludeConfig: makeTab({ selectedLabels: { "label 1": true } }),
    });

    expect(screen.getByRole("checkbox", { name: "label 1" })).toHaveAttribute(
      "aria-disabled",
      "true"
    );
    expect(screen.getByRole("checkbox", { name: "label 2" })).toHaveAttribute(
      "aria-disabled",
      "false"
    );
  });

  it("renders the empty state and triggers onAddLabel when there are no labels", () => {
    const onAddLabel = jest.fn();
    renderSelector({ labels: [], onAddLabel });

    expect(screen.getByText("No labels")).toBeVisible();
    expect(
      screen.getByText("Add a label to target a group of hosts.")
    ).toBeVisible();

    fireEvent.click(screen.getByRole("button", { name: "Add label" }));
    expect(onAddLabel).toHaveBeenCalledTimes(1);
  });
});
