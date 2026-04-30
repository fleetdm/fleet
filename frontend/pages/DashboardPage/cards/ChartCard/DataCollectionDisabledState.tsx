import React from "react";
import { Link } from "react-router";

import paths from "router/paths";

const baseClass = "data-collection-disabled-state";

interface IDataCollectionDisabledStateProps {
  datasetLabel: string;
}

const DataCollectionDisabledState = ({
  datasetLabel,
}: IDataCollectionDisabledStateProps): JSX.Element => {
  return (
    <div className={baseClass}>
      <h3>Data collection is disabled</h3>
      <p>
        <strong>{datasetLabel}</strong> data is not being collected for this
        view.
      </p>
      <p>
        <Link to={paths.ADMIN_ORGANIZATION_ADVANCED}>
          Manage data collection in Advanced settings
        </Link>
      </p>
    </div>
  );
};

export default DataCollectionDisabledState;
