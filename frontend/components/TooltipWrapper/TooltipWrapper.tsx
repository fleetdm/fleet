import classnames from "classnames";
import React from "react";
import ReactTooltip from "react-tooltip";

import { uniqueId } from "lodash";
import { COLORS } from "styles/var/colors";

interface ITooltipWrapperProps {
  children: string;
  tipContent: string;
  position?: "top" | "bottom";
  delayHide?: number;
  underline?: boolean;
  className?: string;
  positionOverrides?: {
    leftAdj?: number;
    topAdj?: number;
  };
}

const baseClass = "component__tooltip-wrapper";

const TooltipWrapper = ({
  children,
  tipContent,
  position = "bottom",
  delayHide,
  underline = true,
  className,
  // positionOverrides = { leftAdj: 54, topAdj: -3 },
  positionOverrides,
}: ITooltipWrapperProps): JSX.Element => {
  const classes = classnames(baseClass, className);
  const tipId = uniqueId();

  const [leftAdj, topAdj] = [
    positionOverrides?.leftAdj ?? 0,
    positionOverrides?.topAdj ?? 0,
  ];

  return (
    <span className={classes}>
      <span
        className={`${baseClass}__element, ${baseClass}__underline`}
        data-tip
        data-for={tipId}
      >
        {children}
        {/* {underline && (
          <span className={`${baseClass}__underline`} data-text={children} />
        )} */}
      </span>
      <ReactTooltip
        className={`${baseClass}__tip-text`}
        place={position ?? "top"}
        type="dark"
        effect="solid"
        id={tipId}
        backgroundColor={COLORS["tooltip-bg"]}
        delayHide={delayHide}
        // delayUpdate={500}
        overridePosition={(pos: { left: number; top: number }) => {
          return {
            left: pos.left + leftAdj,
            top: pos.top + topAdj,
          };
        }}
      >
        {tipContent}
      </ReactTooltip>
    </span>
  );
};

export default TooltipWrapper;
