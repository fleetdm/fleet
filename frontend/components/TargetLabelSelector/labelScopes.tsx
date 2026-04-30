import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";

export type LabelScope =
  | "labelsIncludeAny"
  | "labelsIncludeAll"
  | "labelsExcludeAny";

type LabelScopeEntity = "policy" | "report";

const ENTITY_NOUN: Record<LabelScopeEntity, string> = {
  policy: "Policy",
  report: "Report",
};

interface IGetCustomTargetOptionsArgs {
  entity: LabelScopeEntity;
  isPremiumTier: boolean | undefined;
}

// getCustomTargetOptions returns the label-scope options for the "Custom"
// target dropdown. The "Include all" option is premium-only. The
// "Exclude any" option is policy-only — reports do not support it.
export const getCustomTargetOptions = ({
  entity,
  isPremiumTier,
}: IGetCustomTargetOptionsArgs): IDropdownOption[] => {
  const noun = ENTITY_NOUN[entity];
  const includeAny: LabelScope = "labelsIncludeAny";
  const includeAll: LabelScope = "labelsIncludeAll";
  const excludeAny: LabelScope = "labelsExcludeAny";
  const options: IDropdownOption[] = [
    {
      value: includeAny,
      label: "Include any",
      helpText: (
        <>
          {noun} will target hosts that <b>have any</b> of these labels:
        </>
      ),
      disabled: false,
    },
  ];
  if (isPremiumTier) {
    options.push({
      value: includeAll,
      label: "Include all",
      helpText: (
        <>
          {noun} will target hosts that <b>have all</b> of these labels:
        </>
      ),
      disabled: false,
    });
  }
  if (entity === "policy") {
    options.push({
      value: excludeAny,
      label: "Exclude any",
      helpText: (
        <>
          {noun} will target hosts that <b>don&rsquo;t have any</b> of these
          labels:
        </>
      ),
      disabled: false,
    });
  }
  return options;
};
