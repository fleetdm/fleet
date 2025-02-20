import React, { useContext } from "react";
import { Link } from "react-router";
import classnames from "classnames";

import { AppContext } from "context/app";
import { IConfig } from "interfaces/config";
import { API_ALL_TEAMS_ID, APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import { IUser } from "interfaces/user";
import { QueryParams } from "utilities/url";

import LinkWithContext from "components/LinkWithContext";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";

import UserMenu from "../UserMenu";
import getNavItems, { INavItem } from "./navItems";

interface ISiteTopNavProps {
  config: IConfig;
  currentUser: IUser;
  location: {
    pathname: string;
    query: QueryParams;
  };
  onLogoutUser: () => void;
  onUserMenuItemClick: (path: string) => void;
}

// TODO(sarah): Build RegExps for other routes that need to be differentiated in order to build
// top nav links that match the expected UX.

const REGEX_DETAIL_PAGES = {
  HOST_DETAILS: /\/hosts\/\d+/i,
  LABEL_EDIT: /\/labels\/\d+/i,
  LABEL_NEW: /\/labels\/new/i,
  PACK_EDIT: /\/packs\/\d+/i,
  PACK_NEW: /\/packs\/new/i,
  QUERIES_EDIT: /\/queries\/\d+/i,
  QUERIES_NEW: /\/queries\/new/i,
  POLICY_EDIT: /\/policies\/\d+/i,
  POLICY_NEW: /\/policies\/new/i,
  SOFTWARE_TITLES_DETAILS: /\/software\/titles\/\d+/i,
  SOFTWARE_VERSIONS_DETAILS: /\/software\/versions\/\d+/i,
};

const REGEX_GLOBAL_PAGES = {
  MANAGE_PACKS: /\/packs\/manage/i,
  ORGANIZATION: /\/settings\/organization/i,
  USERS: /\/settings\/users/i,
  INTEGRATIONS: /\/settings\/integrations/i,
  TEAMS: /\/settings\/teams$/i, // Note: we want this to only match if it is the end of the path
  PROFILE: /\/profile/i,
};

const testDetailPage = (path: string, re: RegExp) => {
  if (re === REGEX_DETAIL_PAGES.LABEL_EDIT) {
    // we want to match "/labels/10" but not "/hosts/manage/labels/10"
    return path.match(re) && !path.match(/\/hosts\/manage\/labels\/\d+/); // we're using this approach because some browsers don't support regexp negative lookbehind
  }
  return path.match(re);
};

const isDetailPage = (path: string) => {
  return Object.values(REGEX_DETAIL_PAGES).some((re) =>
    testDetailPage(path, re)
  );
};

const isGlobalPage = (path: string) => {
  return Object.values(REGEX_GLOBAL_PAGES).some((re) => path.match(re));
};

const SiteTopNav = ({
  config,
  currentUser,
  location: { pathname: currentPath, query },
  onLogoutUser,
  onUserMenuItemClick,
}: ISiteTopNavProps): JSX.Element => {
  const {
    currentTeam,
    isAnyTeamAdmin,
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainer,
    isNoAccess,
  } = useContext(AppContext);

  const isActiveDetailPage = isDetailPage(currentPath);
  const isActiveGlobalPage = isGlobalPage(currentPath);

  const currentQueryParams = { ...query };
  if (isActiveGlobalPage || isActiveDetailPage) {
    // detail pages (e.g., host details) and some manage pages (e.g., queries) aren't guaranteed to
    // have a team_id in the URL that we can simply append to the top nav links so instead we need grab the team
    // id from context
    currentQueryParams.team_id =
      currentTeam?.id === APP_CONTEXT_ALL_TEAMS_ID
        ? API_ALL_TEAMS_ID
        : currentTeam?.id;
  }

  const renderNavItem = (navItem: INavItem) => {
    const { name, iconName, withParams } = navItem;
    const orgLogoURL = config.org_info.org_logo_url;
    const active = navItem.location.regex.test(currentPath);

    const navItemBaseClass = "site-nav-item";

    const navItemClasses = classnames(`${navItemBaseClass}`, {
      [`${navItemBaseClass}--active`]: active,
    });

    if (iconName && iconName === "logo") {
      return (
        <li className={navItemClasses} key={`nav-item-${name}`}>
          <LinkWithContext
            className={`${navItemBaseClass}__logo-wrapper`}
            currentQueryParams={currentQueryParams}
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

    if (active && !isActiveDetailPage) {
      const path = navItem.alwaysToPathname
        ? navItem.location.pathname
        : currentPath;

      const includeTeamId = (activePath: string) => {
        if (currentQueryParams.team_id !== API_ALL_TEAMS_ID) {
          return `${path}?team_id=${currentQueryParams.team_id}`;
        }
        return activePath;
      };

      // Clicking an active link returns user to default page
      // Resetting all filters except team ID
      // TODO: Find best pattern(one that doesn't dispatch a
      // replace to the same url, which triggers a re-render)
      return (
        <li className={navItemClasses} key={`nav-item-${name}`}>
          <a className={`${navItemBaseClass}__link`} href={includeTeamId(path)}>
            <span
              className={`${navItemBaseClass}__name`}
              data-text={navItem.name}
            >
              {name}
            </span>
          </a>
        </li>
      );
    }

    return (
      <li className={navItemClasses} key={`nav-item-${name}`}>
        {withParams ? (
          <LinkWithContext
            className={`${navItemBaseClass}__link`}
            withParams={withParams}
            currentQueryParams={{ team_id: currentQueryParams.team_id }}
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

  const userNavItems = getNavItems(
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
          onUserMenuItemClick={onUserMenuItemClick}
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
