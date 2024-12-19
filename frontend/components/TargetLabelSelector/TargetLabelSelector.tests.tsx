import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";

import TargetLabelSelector from "./TargetLabelSelector";

describe("TargetLabelSelector component", () => {
  it("renders the custom target selector when the target type is 'Custom'", () => {
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
});
