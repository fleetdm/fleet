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
}

const baseClass = "component__new-tooltip-wrapper";

const TooltipWrapper = ({
  children,
  tipContent,
  position = "bottom",
  delayHide,
  underline = true,
  className,
}: // positionOverrides = { leftAdj: 54, topAdj: -3 },
ITooltipWrapperProps): JSX.Element => {
  const wrapperClasses = classnames(baseClass, className);
  const elementClasses = classnames(`${baseClass}__element`, {
    [`${baseClass}__underline`]: underline,
  });
  const tipId = uniqueId();

  return (
    <span className={wrapperClasses}>
      <div className={elementClasses} data-tip data-for={tipId}>
        {children}
        {/* {underline && (
          <span className={`${baseClass}__underline`} data-text={children} />
        )} */}
      </div>
      <ReactTooltip
        className={`${baseClass}__tip-text`}
        place={position}
        type="dark"
        effect="solid"
        id={tipId}
        backgroundColor={COLORS["tooltip-bg"]}
        delayHide={delayHide}
      >
        {tipContent}
      </ReactTooltip>
    </span>
  );
};

export default TooltipWrapper;
