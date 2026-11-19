import React, { ReactNode } from "react";

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
// target dropdown.
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

// getLabelsRequiredError returns the inline error text that replaces the
// scope help text when the user clicks Save without selecting any label.
export const getLabelsRequiredError = (scope: LabelScope): ReactNode => {
  switch (scope) {
    case "labelsIncludeAll":
      return (
        <>
          Select at least one label to target hosts that <b>have all</b> of
          these labels:
        </>
      );
    case "labelsExcludeAny":
      return (
        <>
          Select at least one label to target hosts that{" "}
          <b>don&rsquo;t have any</b> of these labels:
        </>
      );
    case "labelsIncludeAny":
    default:
      return (
        <>
          Select at least one label to target hosts that <b>have any</b> of
          these labels:
        </>
      );
  }
};
