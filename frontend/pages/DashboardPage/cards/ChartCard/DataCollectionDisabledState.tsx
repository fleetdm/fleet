import React from "react";
import { browserHistory } from "react-router";

import paths from "router/paths";

import Button from "components/buttons/Button";

const baseClass = "data-collection-disabled-state";

interface IDataCollectionDisabledStateProps {
  datasetLabel: string;
  currentTeamId?: number;
}

const DataCollectionDisabledState = ({
  datasetLabel,
  currentTeamId,
}: IDataCollectionDisabledStateProps): JSX.Element => {
  return (
    <div className={baseClass}>
      <h3>Data collection is disabled</h3>
      <p>
        Turn on &ldquo;{datasetLabel}&rdquo; to see data for{" "}
        {currentTeamId ? `this fleet` : `all fleets`}.
      </p>
      <Button
        onClick={() => {
          currentTeamId
            ? browserHistory.push(paths.FLEET_DETAILS_SETTINGS(currentTeamId))
            : browserHistory.push(paths.ADMIN_ORGANIZATION_ADVANCED);
        }}
      >
        Turn on
      </Button>
    </div>
  );
};

export default DataCollectionDisabledState;
