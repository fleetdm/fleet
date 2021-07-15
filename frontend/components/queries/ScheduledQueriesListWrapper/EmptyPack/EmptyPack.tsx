import React from "react";

const baseClass = "empty-pack";

const EmptyPack = (): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        <div className={`${baseClass}__empty-filter-results`}>
          <h1>PLEASE FILL OUT WITH ACTUAL EMPTY STATE</h1>
        </div>
      </div>
    </div>
  );
};

export default EmptyPack;
