import React, { useContext } from "react";
import { browserHistory } from "react-router";

import Button from "components/buttons/Button";
import { AppContext } from "context/app";
import paths from "router/paths";
import permissions from "utilities/permissions";

const baseClass = "data-collection-disabled-state";

interface IDataCollectionDisabledStateProps {
  datasetLabel: string;
  currentTeamId?: number;
}

const DataCollectionDisabledState = ({
  datasetLabel,
  currentTeamId,
}: IDataCollectionDisabledStateProps): JSX.Element => {
  const { currentUser, isGlobalAdmin } = useContext(AppContext);
  // Resolve team-admin against the fleet we're rendering for, not the app's
  // currently selected fleet. When a host detail page is opened by URL, the
  // app context's `currentTeam` isn't set to the host's fleet.
  const canAccessSettings = currentTeamId
    ? !!(isGlobalAdmin || permissions.isTeamAdmin(currentUser, currentTeamId))
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
