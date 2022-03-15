import React from "react";
import { Link } from "react-router";
import classnames from "classnames";

import { IUser } from "interfaces/user";
import { IConfig } from "interfaces/config";

// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";

import navItems, { INavItem } from "./navItems";

interface IFleetDesktopTopNavProps {
  onLogoutUser: () => void;
  onNavItemClick: (path: string) => void;
  pathname: string;
  currentUser: IUser;
  config: IConfig;
}

const FleetDesktopTopNav = ({
  pathname,
  currentUser,
  config,
}: IFleetDesktopTopNavProps): JSX.Element => {
  const renderNavItem = (navItem: INavItem) => {
    const { name, iconName, withContext } = navItem;
    const orgLogoURL = config.org_logo_url;
    const active = navItem.location.regex.test(pathname);

    const navItemBaseClass = "site-nav-item";

    const navItemClasses = classnames(`${navItemBaseClass}`, {
      [`${navItemBaseClass}--active`]: active,
    });

    return (
      <li className={navItemClasses} key={`nav-item-${name}`}>
        <Link
          className={`${navItemBaseClass}__logo`}
          to={navItem.location.pathname}
        >
          <OrgLogoIcon className="logo" src={orgLogoURL} />
        </Link>
      </li>
    );
  };

  const userNavItems = navItems(currentUser);

  const renderNavItems = () => {
    return (
      <div className="site-nav-container">
        <ul className="site-nav-list">
          {userNavItems.map((navItem) => {
            return renderNavItem(navItem);
          })}
        </ul>
      </div>
    );
  };

  return renderNavItems();
};

export default FleetDesktopTopNav;
