import React from "react";

import Icon from "components/Icon/Icon";

const baseClass = "learn-fleet";

const LearnFleet = (): JSX.Element => {
  return (
    <div className={baseClass}>
      <p>
        Want to explore Fleet&apos;s features? Learn how to ask questions about
        your device using queries.
      </p>
      <a
        className="dashboard-info-card__action-button"
        href="https://fleetdm.com/docs/using-fleet/learn-how-to-use-fleet"
        target="_blank"
        rel="noopener noreferrer"
      >
        Learn how to use Fleet
        <Icon name="arrow-internal-link" color="ui-fleet-black-75" />
      </a>
    </div>
  );
};

export default LearnFleet;
