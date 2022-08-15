import React from "react";
import Spinner from "components/Spinner";

const baseClass = "button-loading-spinner";

interface IButtonSpinnerProps {
  whiteButton?: boolean;
}
const ButtonSpinner = ({ whiteButton }: IButtonSpinnerProps): JSX.Element => {
  if (whiteButton) {
    return <Spinner />;
  }

  return (
    <div className={baseClass}>
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

export default ButtonSpinner;
