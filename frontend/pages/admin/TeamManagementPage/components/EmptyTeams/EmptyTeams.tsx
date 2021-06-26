import React from "react";
import OpenNewTabIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";

const baseClass = "empty-teams";

const EmptyUsers = (): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        <div className={`${baseClass}__empty-filter-results`}>
          <h1>Set up team permissions</h1>
          <p>
            Keep your organization organized and efficient by ensuring every
            user has the correct access to the right hosts.
          </p>
          <p className={"learn-more"}>
            Want to learn more?
            <a
              href="https://github.com/fleetdm/fleet/pull/472"
              target="_blank"
              rel="noopener noreferrer"
            >
              Read about teams
              <img src={OpenNewTabIcon} alt="open new tab" />
            </a>
          </p>
        </div>
      </div>
    </div>
  );
};

export default EmptyUsers;
