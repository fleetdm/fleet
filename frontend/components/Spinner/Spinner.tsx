import React from "react";

export interface ISpinnerProps {
  isInButton?: boolean;
}

const baseClass = "loading-spinner";

const Spinner = ({ isInButton }: ISpinnerProps): JSX.Element => {
  if (isInButton) {
    return (
      <div className="ring ring-for-button">
        <div />
        <div />
        <div />
        <div />
      </div>
    );
  }

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__ring`}>
        <div />
        <div />
        <div />
        <div />
      </div>
    </div>
  );
};

export default Spinner;
