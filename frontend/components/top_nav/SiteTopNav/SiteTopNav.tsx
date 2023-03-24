import React, { useContext } from "react";
import { Link } from "react-router";
import classnames from "classnames";

import { IUser } from "interfaces/user";
import { IConfig } from "interfaces/config";
import { AppContext } from "context/app";
import { QueryParams } from "utilities/url";

import LinkWithContext from "components/LinkWithContext";
import UserMenu from "components/top_nav/UserMenu";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";

import navItems, { INavItem } from "./navItems";

interface ISiteTopNavProps {
  onLogoutUser: () => void;
  onNavItemClick: (path: string) => void;
  pathname: string;
  query: QueryParams;
  currentUser: IUser;
  config: IConfig;
}

const SiteTopNav = ({
  onLogoutUser,
  onNavItemClick,
  pathname,
  query,
  currentUser,
  config,
}: ISiteTopNavProps): JSX.Element => {
  const {
    isAnyTeamAdmin,
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainer,
    isNoAccess,
    isMdmFeatureFlagEnabled,
  } = useContext(AppContext);

  const renderNavItem = (navItem: INavItem) => {
    const { name, iconName, withUrlQueryParams } = navItem;
    const orgLogoURL = config.org_info.org_logo_url;
    const active = navItem.location.regex.test(pathname);

    const navItemBaseClass = "site-nav-item";

    const navItemClasses = classnames(`${navItemBaseClass}`, {
      [`${navItemBaseClass}--active`]: active,
    });

    if (iconName && iconName === "logo") {
      return (
        <li className={navItemClasses} key={`nav-item-${name}`}>
          <Link
            className={`${navItemBaseClass}__logo-wrapper`}
            to={navItem.location.pathname}
          >
            <div className={`${navItemBaseClass}__logo`}>
              <OrgLogoIcon className="logo" src={orgLogoURL} />
            </div>
          </Link>
        </li>
      );
    }

    return (
      <li className={navItemClasses} key={`nav-item-${name}`}>
        {withUrlQueryParams?.length ? (
          <LinkWithContext
            className={`${navItemBaseClass}__link`}
            withUrlQueryParams={withUrlQueryParams}
            query={query}
            to={navItem.location.pathname}
          >
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
    isNoAccess,
    isMdmFeatureFlagEnabled
  );

  const renderNavItems = () => {
    return (
      <div className="site-nav-content">
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
