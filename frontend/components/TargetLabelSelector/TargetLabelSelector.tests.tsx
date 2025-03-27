import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";

import TargetLabelSelector from "./TargetLabelSelector";

describe("TargetLabelSelector component", () => {
  describe("renders the custom target selector when the target type is 'Custom'", () => {
    it("with a dropdown when there are custom options to choose from", () => {
      render(
        <TargetLabelSelector
          selectedTargetType="Custom"
          selectedCustomTarget="labelIncludeAny"
          customTargetOptions={[
            { value: "labelIncludeAny", label: "Include any" },
          ]}
          selectedLabels={{}}
          labels={[
            { id: 1, name: "label 1", label_type: "regular" },
            { id: 2, name: "label 2", label_type: "regular" },
          ]}
          onSelectCustomTarget={noop}
          onSelectLabel={noop}
          onSelectTargetType={noop}
        />
      );

      // custom target selector is rendering
      expect(screen.getByRole("option", { name: "Include any" })).toBeVisible();

      // lables are rendering
      expect(screen.getByRole("checkbox", { name: "label 1" })).toBeVisible();
      expect(screen.getByRole("checkbox", { name: "label 2" })).toBeVisible();
    });

    it("with an optional message and no dropdown when there are no custom options to choose from", () => {
      const HELP_TEXT = "go boldly where no target has gone before";
      render(
        <TargetLabelSelector
          selectedTargetType="Custom"
          selectedCustomTarget="labelIncludeAny"
          customHelpText={<span>{HELP_TEXT}</span>}
          selectedLabels={{}}
          labels={[
            { id: 1, name: "label 1", label_type: "regular" },
            { id: 2, name: "label 2", label_type: "regular" },
          ]}
          onSelectCustomTarget={noop}
          onSelectLabel={noop}
          onSelectTargetType={noop}
        />
      );

      // custom target help text is visible
      expect(screen.getByText(HELP_TEXT)).toBeVisible();

      expect(screen.queryByRole("option")).not.toBeInTheDocument();

      // lables are rendering
      expect(screen.getByRole("checkbox", { name: "label 1" })).toBeVisible();
      expect(screen.getByRole("checkbox", { name: "label 2" })).toBeVisible();
    });
  });

  it("does not render the custom target selector when the target type is 'All hosts'", () => {
    render(
      <TargetLabelSelector
        selectedTargetType="All hosts"
        selectedCustomTarget="labelIncludeAny"
        customTargetOptions={[
          { value: "labelIncludeAny", label: "Include any" },
        ]}
        selectedLabels={{}}
        labels={[
          { id: 1, name: "label 1", label_type: "regular" },
          { id: 2, name: "label 2", label_type: "regular" },
        ]}
        onSelectCustomTarget={noop}
        onSelectLabel={noop}
        onSelectTargetType={noop}
      />
    );

    // custom target selector is not rendering
    expect(screen.queryByRole("option", { name: "Include any" })).toBeNull();

    // lables are not rendering
    expect(screen.queryByRole("checkbox", { name: "label 1" })).toBeNull();
    expect(screen.queryByRole("checkbox", { name: "label 2" })).toBeNull();
  });

  it("renders selected labels as checked", () => {
    render(
      <TargetLabelSelector
        selectedTargetType="Custom"
        selectedCustomTarget="labelIncludeAny"
        customTargetOptions={[
          { value: "labelIncludeAny", label: "Include any" },
        ]}
        selectedLabels={{ "label 1": true, "label 2": false }}
        labels={[
          { id: 1, name: "label 1", label_type: "regular" },
          { id: 2, name: "label 2", label_type: "regular" },
        ]}
        onSelectCustomTarget={noop}
        onSelectLabel={noop}
        onSelectTargetType={noop}
      />
    );

    // lables are rendering
    expect(screen.getByRole("checkbox", { name: "label 1" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "label 2" })).not.toBeChecked();
  });

  it("sets the title to Target by default", () => {
    const TITLE = "Target";
    render(
      <TargetLabelSelector
        selectedTargetType="Custom"
        selectedCustomTarget="labelIncludeAny"
        customTargetOptions={[
          { value: "labelIncludeAny", label: "Include any" },
        ]}
        selectedLabels={{}}
        labels={[
          { id: 1, name: "label 1", label_type: "regular" },
          { id: 2, name: "label 2", label_type: "regular" },
        ]}
        onSelectCustomTarget={noop}
        onSelectLabel={noop}
        onSelectTargetType={noop}
        title={TITLE}
      />
    );

    expect(screen.getByText(TITLE)).toBeVisible();
  });

  it("allows a custom title to be passed in", () => {
    const TITLE = "Choose a target";
    render(
      <TargetLabelSelector
        selectedTargetType="Custom"
        selectedCustomTarget="labelIncludeAny"
        customTargetOptions={[
          { value: "labelIncludeAny", label: "Include any" },
        ]}
        selectedLabels={{}}
        labels={[
          { id: 1, name: "label 1", label_type: "regular" },
          { id: 2, name: "label 2", label_type: "regular" },
        ]}
        onSelectCustomTarget={noop}
        onSelectLabel={noop}
        onSelectTargetType={noop}
        title={TITLE}
      />
    );

    expect(screen.getByText(TITLE)).toBeVisible();
  });

  it("suppresses the title when suppressTitle is true", () => {
    render(
      <TargetLabelSelector
        selectedTargetType="Custom"
        selectedCustomTarget="labelIncludeAny"
        customTargetOptions={[
          { value: "labelIncludeAny", label: "Include any" },
        ]}
        selectedLabels={{}}
        labels={[
          { id: 1, name: "label 1", label_type: "regular" },
          { id: 2, name: "label 2", label_type: "regular" },
        ]}
        onSelectCustomTarget={noop}
        onSelectLabel={noop}
        onSelectTargetType={noop}
        suppressTitle
      />
    );

    expect(screen.queryByText("Target")).not.toBeInTheDocument();
  });
});
