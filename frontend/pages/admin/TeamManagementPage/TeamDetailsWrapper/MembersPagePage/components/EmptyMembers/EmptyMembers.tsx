import React from "react";

const baseClass = "empty-members";

const EmptyMembers = (): JSX.Element => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__inner`}>
        <div className={`${baseClass}__empty-filter-results`}>
          <h1>We couldn&apos;t find any members.</h1>
          <p>
            Expecting to see new members? Try again in a few seconds as the
            system catches up.
          </p>
        </div>
      </div>
    </div>
  );
};

export default EmptyMembers;
