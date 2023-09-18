import React from "react";

import EmptyTable from "components/EmptyTable/EmptyTable";

const AwaitingResults = () => {
  return (
    <EmptyTable
      iconName="collecting-results"
      header="Phoning home..."
      info=" There are currently no results to your query. Please wait while we talk
        to more hosts."
    />
  );
};

export default AwaitingResults;
