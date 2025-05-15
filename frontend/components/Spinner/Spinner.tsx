import React from "react";
import classnames from "classnames";

type Size = "x-small" | "small" | "medium";
type PaddingSize = "small" | "medium";

interface ISpinnerProps {
  small?: boolean;
  button?: boolean;
  white?: boolean;
  /** The size of the spinner. defaults: `"medium"` */
  size?: Size;
  /** The size of the spinner padding. `"medium"` 120px (default), `"small"` 60px */
  verticalPadding?: PaddingSize;
  /** Include the background container styling for the spinner. defaults: `true` */
  includeContainer?: boolean;
  /** Center the spinner in its parent. defaults: `true` */
  centered?: boolean;
  className?: string;
}

const Spinner = ({
  small,
  button,
  white,
  size = "medium",
  verticalPadding = "medium",
  includeContainer = true,
  centered = true,
  className,
}: ISpinnerProps): JSX.Element => {
  const classOptions = classnames(`loading-spinner`, className, size, {
    small,
    button,
    white,
    centered,
    "small-padding": verticalPadding === "small",
    "include-container": includeContainer,
  });
  return (
    <div className={classOptions} data-testid="spinner">
      <div className="loader">
        <svg className="circular" viewBox="25 25 50 50">
          <circle
            className="background"
            cx="50"
            cy="50"
            r="20"
            fill="none"
            strokeWidth="6"
            strokeMiterlimit="10"
          />
          <circle
            className="path"
            cx="50"
            cy="50"
            r="20"
            fill="none"
            strokeWidth="6"
            strokeMiterlimit="10"
          />
        </svg>
      </div>
    </div>
  );
};

export default Spinner;
