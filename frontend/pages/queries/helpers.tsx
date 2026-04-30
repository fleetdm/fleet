import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";

const labelsIncludeAnyOption: IDropdownOption = {
  value: "labelsIncludeAny",
  label: "Include any",
  helpText: (
    <>
      Report will target hosts that <b>have any</b> of these labels:
    </>
  ),
  disabled: false,
};

const labelsIncludeAllOption: IDropdownOption = {
  value: "labelsIncludeAll",
  label: "Include all",
  helpText: (
    <>
      Report will target hosts that <b>have all</b> these labels:
    </>
  ),
  disabled: false,
};

// getQueryCustomTargetOptions returns the report label-scope options for the
// "Custom" target dropdown. The "Include all" option is premium-only.
const getQueryCustomTargetOptions = (
  isPremiumTier: boolean | undefined
): IDropdownOption[] => {
  const options: IDropdownOption[] = [labelsIncludeAnyOption];
  if (isPremiumTier) {
    options.push(labelsIncludeAllOption);
  }
  return options;
};

export default getQueryCustomTargetOptions;
