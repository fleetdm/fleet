import React from "react";

import EmptyState from "components/EmptyState";

const OSVersionsEmptyState = () => {
  return (
    <EmptyState
      header="No OS versions detected"
      info={
        <>
          This report is updated every hour to protect
          <br /> the performance of your devices.
        </>
      }
    />
  );
};

export default OSVersionsEmptyState;
