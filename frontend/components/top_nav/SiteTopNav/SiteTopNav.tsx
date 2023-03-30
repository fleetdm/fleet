import React, { useContext } from "react";
import { Link } from "react-router";
import { Params } from "react-router/lib/Router";
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
  config: IConfig;
  currentUser: IUser;
  location: {
    pathname: string;
    search: string;
    hash?: string;
    query: QueryParams;
  };
  onLogoutUser: () => void;
  onNavItemClick: (path: string) => void;
}

const SiteTopNav = ({
  config,
  currentUser,
  location: { pathname, search, hash = "", query: queryParams },
  onLogoutUser,
  onNavItemClick,
}: ISiteTopNavProps): JSX.Element => {
  const {
    isAnyTeamAdmin,
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainer,
    isNoAccess,
    isMdmEnabledAndConfigured, // TODO: confirm
  } = useContext(AppContext);

  const renderNavItem = (navItem: INavItem) => {
    const { name, iconName, withParams } = navItem;
    const orgLogoURL = config.org_info.org_logo_url;
    const active = navItem.location.regex.test(pathname);

    const navItemBaseClass = "site-nav-item";

    const navItemClasses = classnames(`${navItemBaseClass}`, {
      [`${navItemBaseClass}--active`]: active,
    });

    if (iconName && iconName === "logo") {
      return (
        <li className={navItemClasses} key={`nav-item-${name}`}>
          <LinkWithContext
            className={`${navItemBaseClass}__logo-wrapper`}
            currentQueryParams={queryParams}
            to={navItem.location.pathname}
            withParams={{ type: "query", names: ["team_id"] }}
          >
            <div className={`${navItemBaseClass}__logo`}>
              <OrgLogoIcon className="logo" src={orgLogoURL} />
            </div>
          </LinkWithContext>
        </li>
      );
    }

    if (active) {
      // TODO: confirm link should be noop and find best pattern (one that doesn't dispatch a
      // replace to the same url, which triggers a re-render)
      return (
        <li className={navItemClasses} key={`nav-item-${name}`}>
          <Link
            className={`${navItemBaseClass}__link`}
            to={pathname.concat(search).concat(hash)}
          >
            <span
              className={`${navItemBaseClass}__name`}
              data-text={navItem.name}
            >
              {name}
            </span>
          </Link>
          {/* <div className={`${navItemBaseClass}__link`}>
            <span className={`${navItemBaseClass}__name`}>{name}</span>
          </div> */}
        </li>
      );
    }

    return (
      <li className={navItemClasses} key={`nav-item-${name}`}>
        {withParams ? (
          <LinkWithContext
            className={`${navItemBaseClass}__link`}
            withParams={withParams}
            currentQueryParams={queryParams}
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
    isNoAccess
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
