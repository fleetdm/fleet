import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";

const CUSTOM_TARGET_OPTIONS: IDropdownOption[] = [
  {
    value: "labelsIncludeAny",
    label: "Include any",
    helpText: (
      <>
        Policy will target hosts on selected platforms that <b>have any</b> of
        these labels:
      </>
    ),
    disabled: false,
  },
  {
    value: "labelsExcludeAny",
    label: "Exclude any",
    helpText: (
      <>
        Policy will target hosts on selected platforms that{" "}
        <b>don&rsquo;t have any</b> of these labels:
      </>
    ),
    disabled: false,
  },
];

export default CUSTOM_TARGET_OPTIONS;
