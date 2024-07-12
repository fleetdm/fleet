import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";

export const CUSTOM_TARGET_OPTIONS: IDropdownOption[] = [
  {
    value: "labelsIncludeAll",
    label: "Include all ",
    helpText: (
      <>
        Profile will only be applied to hosts that have <b>all</b> of these
        labels{" "}
      </>
    ),
    disabled: false,
  },
  {
    value: "labelsExcludeAny",
    label: "Exclude all",
    helpText: (
      <>
        Profile will be applied to hosts that don&apos;t have <b>any</b> of
        these labels{" "}
      </>
    ),
    disabled: false,
  },
];

export const listNamesFromSelectedLabels = (dict: Record<string, boolean>) => {
  return Object.entries(dict).reduce((acc, [labelName, isSelected]) => {
    if (isSelected) {
      acc.push(labelName);
    }
    return acc;
  }, [] as string[]);
};

export type CustomTargetOption = "labelsIncludeAll" | "labelsExcludeAny";

export const generateLabelKey = (
  target: string,
  customTargetOption: CustomTargetOption,
  selectedLabels: Record<string, boolean>
) => {
  if (target !== "Custom") {
    return {};
  }

  return {
    [customTargetOption]: listNamesFromSelectedLabels(selectedLabels),
  };
};
