import React from "react";

import EmptyTable from "components/EmptyTable";

const baseClass = "os-versions-empty-state";

const OSVersionsEmptyState = () => {
  return (
    <EmptyTable
      className={`${baseClass}__empty-table`}
      header="No OS versions detected"
      info={
        <span>
          This report is updated every hour to protect
          <br /> the performance of your devices.
        </span>
      }
    />
  );
};

export default OSVersionsEmptyState;
