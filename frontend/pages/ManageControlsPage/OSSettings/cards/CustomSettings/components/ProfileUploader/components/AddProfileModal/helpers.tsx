import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";

export const CUSTOM_TARGET_OPTIONS: IDropdownOption[] = [
  {
    value: "labelsIncludeAll",
    label: "Include all",
    helpText: (
      <>
        Profile will only be applied to hosts that <b>have all</b> of these
        labels.
      </>
    ),
    disabled: false,
  },
  {
    value: "labelsIncludeAny",
    label: "Include any",
    helpText: (
      <>
        Profile will only be applied to hosts that <b>have any</b> of these
        labels.
      </>
    ),
    disabled: false,
  },
  {
    value: "labelsExcludeAny",
    label: "Exclude any",
    helpText: (
      <>
        Profile will only be applied to hosts that <b>don&apos;t have any</b> of
        these labels.
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

export type CustomTargetOption =
  | "labelsIncludeAll"
  | "labelsIncldeAny"
  | "labelsExcludeAny";

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

export const getDescriptionText = (value: string) => {
  return CUSTOM_TARGET_OPTIONS.find((option) => option.value === value)
    ?.helpText;
};
