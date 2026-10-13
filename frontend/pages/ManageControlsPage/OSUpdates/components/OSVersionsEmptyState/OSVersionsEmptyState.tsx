import React from "react";

import EmptyState from "components/EmptyState";

const OSVersionsEmptyState = () => {
  return (
    <EmptyState
      header="No operating systems detected"
      info="Operating system data will appear after the next scheduled check-in."
    />
  );
};

export default OSVersionsEmptyState;
