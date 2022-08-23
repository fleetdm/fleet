import React from "react";
import classnames from "classnames";

interface ISpinnerProps {
  small?: boolean;
  button?: boolean;
  white?: boolean;
}

const Spinner = ({ small, button, white }: ISpinnerProps): JSX.Element => {
  const classOptions = classnames(`loading-spinner`, {
    small,
    button,
    white,
  });
  return (
    <div className={classOptions}>
      <div className="loader">
        <svg className="circular" viewBox="25 25 50 50">
          <circle
            className="path"
            cx="50"
            cy="50"
            r="20"
            fill="none"
            strokeWidth="5"
            strokeMiterlimit="10"
          />
        </svg>
      </div>
    </div>
  );
};

export default Spinner;
