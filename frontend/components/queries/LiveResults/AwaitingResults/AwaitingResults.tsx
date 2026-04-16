import React from "react";

import EmptyState from "components/EmptyState";

const baseClass = "awaiting-results";

const AwaitingResults = () => {
  return (
    <EmptyState
      graphicName="collecting-results"
      header="Phoning home..."
      info=" There are currently no results to your report. Please wait while we talk
        to more hosts."
      className={baseClass}
    />
  );
};

export default AwaitingResults;
