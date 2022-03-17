import React from "react";
import FleetIcon from "../../../../assets/images/fleet-avatar-24x24@2x.png";

// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";

interface IFleetDesktopTopNavProps {
  orgLogoURL?: string;
}

const FleetDesktopTopNav = ({
  orgLogoURL,
}: IFleetDesktopTopNavProps): JSX.Element => {
  const logo = orgLogoURL || FleetIcon;

  return (
    <div className="site-nav-container">
      <ul className="site-nav-list">
        <li className={`site-nav-item--logo`} key={`nav-item`}>
          <OrgLogoIcon className="logo" src={logo} />
        </li>
      </ul>
    </div>
  );
};

export default FleetDesktopTopNav;
