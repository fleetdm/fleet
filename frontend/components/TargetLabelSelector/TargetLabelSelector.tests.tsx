import {
  render,
  screen,
  fireEvent,
  waitFor,
  within,
} from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import { ILabelSummary } from "interfaces/label";
import { createCustomRenderer } from "test/test-utils";

import TargetLabelSelector, { ILabelConfig } from "./TargetLabelSelector";

// jsdom has no layout, so whether a label name is truncated is controlled here.
jest.mock("hooks/useCheckTruncatedElement", () => ({
  useCheckTruncatedElement: jest.fn(),
}));
const mockedUseCheckTruncatedElement = useCheckTruncatedElement as jest.Mock;

const LABELS: ILabelSummary[] = [
  { id: 1, name: "label 1", label_type: "regular" },
  { id: 2, name: "label 2", label_type: "regular" },
];

const LONG_NAME = "label with a name too long to fit in the label picker";
const LABELS_WITH_LONG_NAME: ILabelSummary[] = [
  ...LABELS,
  { id: 3, name: LONG_NAME, label_type: "regular" },
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
  beforeEach(() => {
    mockedUseCheckTruncatedElement.mockReturnValue(false);
  });

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

  it("shows the GitOps tooltip on a selected label tag when disabled by GitOps mode", async () => {
    const renderWithContext = createCustomRenderer({
      context: {
        app: {
          config: {
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "a.b.cc",
            },
          },
        },
      },
    });

    const { user } = renderWithContext(
      <TargetLabelSelector
        selectedTargetType="Custom"
        onSelectTargetType={noop}
        labels={LABELS}
        includeConfig={makeTab({ selectedLabels: { "label 1": true } })}
        excludeConfig={makeTab()}
        emptyStateDescription="Add a label to target a group of hosts."
        onAddLabel={noop}
        disableOptions
      />
    );

    const dismissButton = screen.getByRole("button", {
      name: "Remove label 1",
    });
    expect(dismissButton).toBeDisabled();

    // Hover the tag itself, not the checkbox that has the same text
    await user.hover(dismissButton.parentElement as HTMLElement);
    await waitFor(() => {
      expect(screen.getByText(/Manage in/)).toBeInTheDocument();
    });
  });

  it("shows a truncated selected label's full name in a tooltip on hover", async () => {
    mockedUseCheckTruncatedElement.mockReturnValue(true);
    const { user } = createCustomRenderer()(
      <TargetLabelSelector
        selectedTargetType="Custom"
        onSelectTargetType={noop}
        labels={LABELS_WITH_LONG_NAME}
        includeConfig={makeTab({ selectedLabels: { [LONG_NAME]: true } })}
        excludeConfig={makeTab()}
        emptyStateDescription="Add a label to target a group of hosts."
        onAddLabel={noop}
      />
    );

    const tag = screen.getByRole("button", { name: `Remove ${LONG_NAME}` })
      .parentElement as HTMLElement;
    await user.hover(within(tag).getByText(LONG_NAME));

    expect(await screen.findByRole("tooltip")).toHaveTextContent(LONG_NAME);
  });

  it("shows no tooltip for a selected label name that fits", async () => {
    const { user } = createCustomRenderer()(
      <TargetLabelSelector
        selectedTargetType="Custom"
        onSelectTargetType={noop}
        labels={LABELS}
        includeConfig={makeTab({ selectedLabels: { "label 1": true } })}
        excludeConfig={makeTab()}
        emptyStateDescription="Add a label to target a group of hosts."
        onAddLabel={noop}
      />
    );

    const tag = screen.getByRole("button", { name: "Remove label 1" })
      .parentElement as HTMLElement;
    await user.hover(within(tag).getByText("label 1"));

    await new Promise((resolve) => setTimeout(resolve, 500));
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });

  it("shows only the GitOps tooltip on a truncated label tag disabled by GitOps mode", async () => {
    mockedUseCheckTruncatedElement.mockReturnValue(true);
    const { user } = createCustomRenderer({
      context: {
        app: {
          config: {
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "a.b.cc",
            },
          },
        },
      },
    })(
      <TargetLabelSelector
        selectedTargetType="Custom"
        onSelectTargetType={noop}
        labels={LABELS_WITH_LONG_NAME}
        includeConfig={makeTab({ selectedLabels: { [LONG_NAME]: true } })}
        excludeConfig={makeTab()}
        emptyStateDescription="Add a label to target a group of hosts."
        onAddLabel={noop}
        disableOptions
      />
    );

    const tag = screen.getByRole("button", { name: `Remove ${LONG_NAME}` })
      .parentElement as HTMLElement;
    await user.hover(within(tag).getByText(LONG_NAME));

    const tooltip = await screen.findByRole("tooltip");
    expect(tooltip).toHaveTextContent(/Manage in/);
    expect(tooltip).not.toHaveTextContent(LONG_NAME);
    expect(screen.getAllByRole("tooltip")).toHaveLength(1);
  });
});
