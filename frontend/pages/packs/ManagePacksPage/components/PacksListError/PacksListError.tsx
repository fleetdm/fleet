/**
 * Component when there is an error retrieving schedule set up in fleet
 */
import React from "react";

import OpenNewTabIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";
import ErrorIcon from "../../../../../../assets/images/icon-error-16x16@2x.png";

const baseClass = "packs-list-error";

const PacksListError = (): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        <div className="info">
          <span className="info__header">
            <img src={ErrorIcon} alt="error icon" id="error-icon" />
            Something&apos;s gone wrong.
          </span>
          <span className="info__data">Refresh the page or log in again.</span>
          <span className="info__data">
            If this keeps happening, please&nbsp;
            <a
              href="https://github.com/fleetdm/fleet/issues"
              target="_blank"
              rel="noopener noreferrer"
            >
              file an issue.
              <img src={OpenNewTabIcon} alt="open new tab" id="new-tab-icon" />
            </a>
          </span>
        </div>
      </div>
    </div>
  );
};

export default PacksListError;
