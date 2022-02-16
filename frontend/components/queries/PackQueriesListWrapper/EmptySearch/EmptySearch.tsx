import React from "react";

const baseClass = "empty-pack";

const EmptyPack = (): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        <div className={`${baseClass}__empty-filter-results`}>
          <h1>No queries matched your search criteria.</h1>
          <p>Try a different search.</p>
        </div>
      </div>
    </div>
  );
};

export default EmptyPack;
