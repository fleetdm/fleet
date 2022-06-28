import React from "react";

const baseClass = "loading-spinner";

const Spinner = (): JSX.Element => {
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
