import React from "react";

import EmptyTable from "components/EmptyTable/EmptyTable";

const baseClass = "awaiting-results";

const AwaitingResults = () => {
  return (
    <EmptyTable
      graphicName="collecting-results"
      header="Phoning home..."
      info=" There are currently no results to your query. Please wait while we talk
        to more hosts."
      className={baseClass}
    />
  );
};

export default AwaitingResults;
