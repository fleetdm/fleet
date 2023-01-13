// This component is used on ManageSoftwarePage.tsx and DashboardPage.tsx > Software.tsx card

import React from "react";

import CustomLink from "components/CustomLink";

const baseClass = "manage-software-page";

type IEmptySoftwareProps = {
  message: "disabled" | "collecting" | "default" | "";
  noSandboxHosts?: boolean;
};

const EmptySoftware = ({
  message,
  noSandboxHosts,
}: IEmptySoftwareProps): JSX.Element => {
  switch (message) {
    case "disabled": {
      return (
        <div className={`${baseClass}__empty-software`}>
          <div className="empty-software__inner">
            <h1>Software inventory is disabled.</h1>
            <p>
              Check out the Fleet documentation on{" "}
              <CustomLink
                url="https://fleetdm.com/docs/using-fleet/vulnerability-processing#configuration"
                text="how to configure software inventory"
                newTab
              />
            </p>
          </div>
        </div>
      );
    }
    case "collecting": {
      return (
        <div className={`${baseClass}__empty-software`}>
          <div className="empty-software__inner">
            <h1>
              {noSandboxHosts
                ? "Fleet begins collecting software inventory after a host is enrolled"
                : "Fleet is collecting software inventory"}
            </h1>
            <p>
              Try again in about{" "}
              {noSandboxHosts
                ? "15 minutes after host enrollment."
                : "1 hour as the system catches up."}
            </p>
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
