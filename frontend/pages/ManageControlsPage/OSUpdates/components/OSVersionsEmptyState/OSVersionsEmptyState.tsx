import React from "react";

import EmptyState from "components/EmptyState";

const baseClass = "os-versions-empty-state";

const OSVersionsEmptyState = () => {
  return (
    <EmptyState
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
