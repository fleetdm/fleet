import React from "react";

export type TooltipInnerContentOption = string | TooltipInnerContentFunc;

export type TooltipInnerContentFunc = (
  innerProps: TooltipInnerContentProps
) => string | JSX.Element;

export type TooltipInnerContentProps = Record<
  string,
  string | number | boolean
>;

export const TooltipContent = (props: {
  innerContent: TooltipInnerContentOption;
  innerProps: TooltipInnerContentProps;
}): JSX.Element => {
  const { innerContent: tooltip, innerProps: args } = props;
  if (typeof tooltip === "function") {
    return <>{tooltip(args)}</>;
  }
  return <>{tooltip}</>;
};
