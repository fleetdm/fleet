import React, { useContext } from "react";
import { browserHistory } from "react-router";

import Button from "components/buttons/Button";
import { AppContext } from "context/app";
import paths from "router/paths";
import permissions from "utilities/permissions";

const baseClass = "data-collection-disabled-state";

interface IDataCollectionDisabledStateProps {
  datasetLabel: string;
  // Org-level historical_data setting for this dataset. When false, the fleet
  // toggle can't take effect and only a global admin can turn it back on.
  globallyEnabled: boolean;
  currentTeamId?: number;
}

const DataCollectionDisabledState = ({
  datasetLabel,
  globallyEnabled,
  currentTeamId,
}: IDataCollectionDisabledStateProps): JSX.Element => {
  const { currentUser, isGlobalAdmin } = useContext(AppContext);
  // Only offer a fleet-level fix when the org gate is open — a team admin
  // can't override a disabled global setting. Resolve team-admin against the
  // fleet we're rendering for, not the app's currently selected fleet.
  const scopedToFleet = !!currentTeamId && globallyEnabled;
  const canAccessSettings = scopedToFleet
    ? !!(isGlobalAdmin || permissions.isTeamAdmin(currentUser, currentTeamId))
    : !!isGlobalAdmin;

  // Scope text tracks the user's context (which fleet they're viewing), not
  // where the fix lives — otherwise a team admin sees "all fleets" while
  // clearly inside one fleet's dashboard.
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
            scopedToFleet
              ? browserHistory.push(
                  paths.FLEET_DETAILS_SETTINGS(currentTeamId as number)
                )
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
