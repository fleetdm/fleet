import React from "react";
import { Link } from "react-router";

// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import FleetIcon from "../../../../assets/images/fleet-avatar-24x24@2x.png";

interface ILogoOnlyNavProps {
  /** When set, the logo links to this path. */
  to?: string;
}

const LogoOnlyNav = ({ to }: ILogoOnlyNavProps) => {
  const logo = (
    <div className="site-nav-item__logo-wrapper">
      <div className="site-nav-item__logo">
        <OrgLogoIcon className="logo" src={FleetIcon} />
      </div>
    </div>
  );

  return (
    <div className="site-nav-content">
      <ul className="site-nav-left">
        <li className="site-nav-item dup-org-logo" key="dup-org-logo">
          {to ? <Link to={to}>{logo}</Link> : logo}
        </li>
      </ul>
    </div>
  );
};

export default LogoOnlyNav;
