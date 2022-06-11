// This component is used on ManageSoftwarePage.tsx and Homepage.tsx > Software.tsx card

import React from "react";

import ExternalLinkIcon from "../../../../assets/images/open-new-tab-12x12@2x.png";

const baseClass = "manage-software-page";

type IEmptySoftware = "disabled" | "collecting" | "default" | "";

const EmptySoftware = (message: IEmptySoftware): JSX.Element => {
  switch (message) {
    case "disabled": {
      return (
        <div className={`${baseClass}__empty-software`}>
          <div className="empty-software__inner">
            <h1>Software inventory is disabled.</h1>
            <p>
              Check out the Fleet documentation on{" "}
              <a
                href="https://fleetdm.com/docs/using-fleet/vulnerability-processing#configuration"
                target="_blank"
                rel="noopener noreferrer"
              >
                how to configure software inventory{" "}
                <img alt="External link" src={ExternalLinkIcon} />
              </a>
            </p>
          </div>
        </div>
      );
    }
    case "collecting": {
      return (
        <div className={`${baseClass}__empty-software`}>
          <div className="empty-software__inner">
            <h1>Fleet is collecting software inventory.</h1>
            <p>Try again in about 1 hour as the system catches up.</p>
          </div>
        </div>
      );
    }
    default: {
      return (
        <div className={`${baseClass}__empty-software`}>
          <div className="empty-software__inner">
            <h1>No software matches the current search criteria.</h1>
            <p>Try again in about 1 hour as the system catches up.</p>
          </div>
        </div>
      );
    }
  }
};

export default EmptySoftware;
