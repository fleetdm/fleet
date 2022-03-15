import React from "react";

import { IUser } from "interfaces/user";
import { IConfig } from "interfaces/config";

// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";

interface IFleetDesktopTopNavProps {
  config: IConfig;
}

const FleetDesktopTopNav = ({
  config,
}: IFleetDesktopTopNavProps): JSX.Element => {
  const orgLogoURL = config.org_logo_url;

  return (
    <div className="site-nav-container">
      <ul className="site-nav-list">
        <li className={`site-nav-item--logo`} key={`nav-item`}>
          <OrgLogoIcon className="logo" src={orgLogoURL} />
        </li>
      </ul>
    </div>
  );
};

export default FleetDesktopTopNav;
