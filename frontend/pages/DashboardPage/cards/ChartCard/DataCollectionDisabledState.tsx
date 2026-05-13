import React, { useContext } from "react";
import { browserHistory } from "react-router";

import { AppContext } from "context/app";
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
  const { isGlobalAdmin, isTeamAdmin } = useContext(AppContext);
  const canAccessSettings = currentTeamId
    ? !!(isGlobalAdmin || isTeamAdmin)
    : !!isGlobalAdmin;

  const scopeText = currentTeamId ? "this fleet" : "all fleets";

  return (
    <div className={baseClass}>
      <h3>Data collection is disabled</h3>
      <p>
        {canAccessSettings ? "Turn on" : "Ask an admin to turn on"} &ldquo;
        {datasetLabel}&rdquo; to see data for {scopeText}.
      </p>
      {canAccessSettings && (
        <Button
          onClick={() => {
            currentTeamId
              ? browserHistory.push(paths.FLEET_DETAILS_SETTINGS(currentTeamId))
              : browserHistory.push(paths.ADMIN_ORGANIZATION_ADVANCED);
          }}
        >
          Turn on
        </Button>
      )}
    </div>
  );
};

export default DataCollectionDisabledState;
