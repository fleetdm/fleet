import React, { useRef } from "react";

import TooltipWrapper from "components/TooltipWrapper";
import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

const baseClass = "user-value";

interface IUserValueProps {
  values: string[];
}

const UserValue = ({ values }: IUserValueProps) => {
  const displayNameRef = useRef<HTMLSpanElement>(null);
  const isTruncated = useCheckTruncatedElement(displayNameRef);

  let content: React.ReactNode = null;

  if (values.length === 0) {
    // no values content
    content = DEFAULT_EMPTY_CELL_VALUE;
  } else if (values.length === 1) {
    // single value content
    content = (
      <TooltipWrapper
        tipContent={values[0]}
        underline={false}
        disableTooltip={!isTruncated}
        position="top"
        showArrow
      >
        <span ref={displayNameRef} className={`${baseClass}__single`}>
          {values[0]}
        </span>
      </TooltipWrapper>
    );
  } else {
    const tipContent = (
      <div className={`${baseClass}__tooltip-list`}>
        {values.map((value) => (
          <div key={value}>{value}</div>
        ))}
      </div>
    );
    content = (
      <TooltipWrapper
        tipContent={tipContent}
        underline={false}
        position="bottom"
        showArrow
      >
        <span className={`${baseClass}__multi-wrapper`}>
          <span ref={displayNameRef} className={`${baseClass}__multi`}>
            {values[0]}
          </span>
          {/* the ellipsis already separates truncated text from "+ N more" */}
          {!isTruncated ? " " : null}
          <span>+ {values.length - 1} more</span>
        </span>
      </TooltipWrapper>
    );
  }

  return <div className={`${baseClass}`}>{content}</div>;
};

export default UserValue;
