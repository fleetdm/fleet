import classNames from "classnames";
import React, { useContext } from "react";
import { InjectedRouter } from "react-router";

import CommandPalette from "components/CommandPalette";
import SiteTopNav from "components/top_nav/SiteTopNav";
import { AppContext } from "context/app";
import UnsupportedScreenSize from "layouts/UnsupportedScreenSize";
import shouldShowUnsupportedScreen from "layouts/UnsupportedScreenSize/helpers";
import paths from "router/paths";
import { QueryParams } from "utilities/url";

interface ICoreLayoutProps {
  children: React.ReactNode;
  router: InjectedRouter; // v3
  // TODO: standardize typing and usage of location across app components
  location: {
    pathname: string;
    search: string;
    hash?: string;
    query: QueryParams;
  };
}

const CoreLayout = ({ children, router, location }: ICoreLayoutProps) => {
  const { config, currentUser } = useContext(AppContext);

  const onLogoutUser = async () => {
    const { LOGOUT } = paths;
    router.push(LOGOUT);
  };

  const onUserMenuItemClick = (path: string) => {
    router.push(path);
  };

  if (!currentUser || !config) {
    return null;
  }

  const coreWrapperClassnames = classNames("core-wrapper", {
    "low-width-supported": !shouldShowUnsupportedScreen(location.pathname),
  });

  return (
    <div className="app-wrap">
      <CommandPalette />
      {shouldShowUnsupportedScreen(location.pathname) && (
        <UnsupportedScreenSize />
      )}
      <nav className="site-nav-container">
        <SiteTopNav
          config={config}
          currentUser={currentUser}
          location={location}
          onLogoutUser={onLogoutUser}
          onUserMenuItemClick={onUserMenuItemClick}
        />
      </nav>
      <div className={coreWrapperClassnames}>{children}</div>
    </div>
  );
};

export default CoreLayout;
