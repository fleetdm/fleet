import React from "react";

interface ISpinnerProps {
  small?: boolean;
}

const baseClass = "loading-spinner";

const Spinner = ({ small }: ISpinnerProps): JSX.Element => {
  return (
    <div className={`${baseClass} ${small ? "small" : ""}`}>
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
