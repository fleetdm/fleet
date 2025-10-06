import React from "react";

import EmptyTable from "components/EmptyTable";
import Card from "components/Card";

const baseClass = "os-versions-empty-state";

const OSVersionsEmptyState = () => {
  return (
    <Card>
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
    </Card>
  );
};

export default OSVersionsEmptyState;
