/**
 * Component when there is no host results found in a search
 */
import React from "react";

const baseClass = "empty-users";

const EmptyUsers = (): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        <div className={`${baseClass}__empty-filter-results`}>
          <h1>No users match the current criteria</h1>
          <p>
            Expecting to see users? Try again in a few seconds as the system
            catches up
          </p>
        </div>
      </div>
    </div>
  );
};

export default EmptyUsers;
