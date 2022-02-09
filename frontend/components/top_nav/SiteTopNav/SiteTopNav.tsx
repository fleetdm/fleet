import React, { useContext } from "react";
import { Link } from "react-router";
import classnames from "classnames";

import { IUser } from "interfaces/user";
import { IConfig } from "interfaces/config";

import LinkWithContext from "components/LinkWithContext";
import UserMenu from "components/top_nav/UserMenu";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";

import { AppContext } from "context/app";

import navItems, { INavItem } from "./navItems";

import HostsIcon from "../../../../assets/images/icon-main-hosts@2x-16x16@2x.png";
import SoftwareIcon from "../../../../assets/images/icon-software-16x16@2x.png";
import QueriesIcon from "../../../../assets/images/icon-main-queries@2x-16x16@2x.png";
import PacksIcon from "../../../../assets/images/icon-main-packs@2x-16x16@2x.png";
import PoliciesIcon from "../../../../assets/images/icon-main-policies-16x16@2x.png";

interface ISiteTopNavProps {
  onLogoutUser: () => any;
  onNavItemClick: () => any;
  pathname: string;
  currentUser: IUser;
  config: IConfig;
}

const SiteTopNav = ({
  onLogoutUser,
  onNavItemClick,
  pathname,
  currentUser,
  config,
}: ISiteTopNavProps): JSX.Element => {
  const {
    isAnyTeamAdmin,
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainer,
    isNoAccess,
  } = useContext(AppContext);

  const renderNavItem = (navItem: INavItem) => {
    const { name, iconName, withContext } = navItem;
    const orgLogoURL = config.org_logo_url;
    const active = navItem.location.regex.test(pathname);

    const navItemBaseClass = "site-nav-item";

    const navItemClasses = classnames(`${navItemBaseClass}`, {
      [`${navItemBaseClass}--active`]: active,
    });

    const iconClasses = classnames([`${navItemBaseClass}__icon`]);

    if (iconName === "logo") {
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
    }

    const iconImage = () => {
      switch (iconName) {
        case "hosts":
          return HostsIcon;
        case "software":
          return SoftwareIcon;
        case "queries":
          return QueriesIcon;
        case "packs":
          return PacksIcon;
        default:
          return PoliciesIcon;
      }
    };

    const icon = (
      <img src={iconImage()} alt={`${iconName} icon`} className={iconClasses} />
    );

    return (
      <li className={navItemClasses} key={`nav-item-${name}`}>
        {withContext ? (
          <LinkWithContext
            className={`${navItemBaseClass}__link`}
            to={navItem.location.pathname}
          >
            {icon}
            <span
              className={`${navItemBaseClass}__name`}
              data-text={navItem.name}
            >
              {name}
            </span>
          </LinkWithContext>
        ) : (
          <Link
            className={`${navItemBaseClass}__link`}
            to={navItem.location.pathname}
          >
            {icon}
            <span
              className={`${navItemBaseClass}__name`}
              data-text={navItem.name}
            >
              {name}
            </span>
          </Link>
        )}
      </li>
    );
  };

  const userNavItems = navItems(
    currentUser,
    isGlobalAdmin,
    isAnyTeamAdmin,
    isAnyTeamMaintainer,
    isGlobalMaintainer,
    isNoAccess
  );

  const renderNavItems = () => {
    return (
      <div className="site-nav-container">
        <ul className="site-nav-list">
          {userNavItems.map((navItem) => {
            return renderNavItem(navItem);
          })}
        </ul>
        <UserMenu
          onLogout={onLogoutUser}
          onNavItemClick={onNavItemClick}
          currentUser={currentUser}
          isAnyTeamAdmin={isAnyTeamAdmin}
          isGlobalAdmin={isGlobalAdmin}
        />
      </div>
    );
  };

  return renderNavItems();
};

export default SiteTopNav;
