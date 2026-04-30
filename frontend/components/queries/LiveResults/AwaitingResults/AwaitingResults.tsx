import React from "react";

import EmptyState from "components/EmptyState";

const baseClass = "awaiting-results";

const AwaitingResults = () => {
  return (
    <EmptyState
      header="Waiting for results"
      info={
        <>
          No hosts have responded yet.
          <br />
          Results will appear as hosts check in.
        </>
      }
      className={baseClass}
    />
  );
};

export default AwaitingResults;
